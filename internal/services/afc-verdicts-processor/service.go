package afcverdictsprocessor

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	clientmessageblockedjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/client-message-blocked"
	clientmessagesentjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/client-message-sent"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

const (
	// serviceName = "afc-verdict-processor".

	statusOk         = `ok`
	statusSuspicious = `suspicious`
)

var (
	ErrProcessingMessageWithKey = errors.New("processing msg with key")
	ErrUnknownStatus            = errors.New("unknown status")
)

//go:generate mockgen -source=$GOFILE -destination=mocks/service_mock.gen.go -package=afcverdictsprocessormocks

type messagesRepository interface {
	MarkAsVisibleForManager(ctx context.Context, msgID types.MessageID) error
	BlockMessage(ctx context.Context, msgID types.MessageID) error
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	backoffInitialInterval time.Duration `default:"100ms" validate:"min=50ms,max=1s"`
	backoffMaxElapsedTime  time.Duration `default:"5s" validate:"min=500ms,max=1m"`
	backoffFactor          float64       `default:"5" validate:"min=1.01,max=10"`

	brokers         []string `option:"mandatory" validate:"min=1"`
	consumers       int      `option:"mandatory" validate:"min=1,max=16"`
	consumerGroup   string   `option:"mandatory" validate:"required"`
	verdictsTopic   string   `option:"mandatory" validate:"required"`
	verdictsSignKey string

	processBatchSize       int           `default:"1" validate:"min=1,max=1000"`
	processBatchMaxTimeout time.Duration `default:"100ms" validate:"min=50ms,max=10s"`
	retries                int           `default:"3" validate:"min=1,max=10"`

	readerFactory KafkaReaderFactory `option:"mandatory" validate:"required"`
	dlqWriter     KafkaDLQWriter     `option:"mandatory" validate:"required"`

	txtor   transactor         `option:"mandatory" validate:"required"`
	msgRepo messagesRepository `option:"mandatory" validate:"required"`
	outBox  outboxService      `option:"mandatory" validate:"required"`
}

type Service struct {
	Options
	publicKey *rsa.PublicKey
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validating afc verdict processor service options: %v", err)
	}
	s := Service{
		Options: opts,
	}
	if opts.verdictsSignKey != "" {
		block, _ := pem.Decode([]byte(opts.verdictsSignKey))
		if block == nil {
			return nil, errors.New("invalid signing key")
		}
		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		publicKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("invalid key type")
		}
		s.publicKey = publicKey
	}
	return &s, nil
}

type verdict struct {
	ChatID    string `json:"chatId"`
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

func (s *Service) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	defer func() {
		if err := s.dlqWriter.Close(); err != nil {
			zap.L().Warn("close dlqWriter", zap.Error(err))
		}
	}()
	for i := 0; i < s.consumers; i++ {
		eg.Go(func() error {
			reader := s.readerFactory(s.brokers, s.consumerGroup, s.verdictsTopic)
			defer func() {
				if err := reader.Close(); err != nil {
					zap.L().Warn("close reader", zap.Error(err))
				}
			}()
			return s.processMessages(ctx, reader)
		})
	}
	return eg.Wait()
}

func (s *Service) processMessages(ctx context.Context, reader KafkaReader) error {
	errCh := make(chan struct{})
	msgCh := make(chan kafka.Message)
	var messages []kafka.Message
	var msg kafka.Message
	go func() {
		var err error
		for {
			msg, err = reader.FetchMessage(ctx)
			if err != nil {
				zap.L().Warn("fetching message", zap.Error(err))
				errCh <- struct{}{}
				break
			}
			msgCh <- msg
		}
	}()
	for {
		messages = messages[:0]
	LOOP:
		for j := 0; j < s.processBatchSize; j++ {
			select {
			case <-time.After(s.processBatchMaxTimeout):
				break LOOP
			case <-errCh:
				return nil
			case msg := <-msgCh:
				messages = append(messages, msg)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			s.processBatch(ctx, messages)
			if err := reader.CommitMessages(ctx, messages...); err != nil {
				zap.L().Warn("CommitMessages", zap.Error(err))
			}
		}
	}
}

func (s *Service) processBatch(ctx context.Context, messages []kafka.Message) {
	for _, msg := range messages {
		v, msgID, err := s.decodeMsg(msg.Value)
		if err == nil {
			for j := 0; j < s.retries; j++ {
				if err = s.processVerdict(ctx, msgID, v); err != nil && !errors.Is(err, ErrUnknownStatus) {
					continue
				}
				break
			}
			if err != nil {
				msg.Headers = append(msg.Headers, formHeaders(err, msg.Partition)...)
			}
		}
		if err != nil {
			msg.Topic = ""
			if err = s.dlqWriter.WriteMessages(ctx, msg); err != nil {
				zap.L().Warn("produce do dlq", zap.Error(err))
			}
		}
	}
}

func formHeaders(lastError error, originalPartition int) []kafka.Header {
	return []kafka.Header{
		{
			Key:   "LAST_ERROR",
			Value: []byte(lastError.Error()),
		},
		{
			Key:   "ORIGINAL_PARTITION",
			Value: []byte(strconv.Itoa(originalPartition)),
		},
	}
}

func (s *Service) processVerdict(ctx context.Context, msgID types.MessageID, v verdict) error {
	switch v.Status {
	case statusOk:
		return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
			if err := s.msgRepo.MarkAsVisibleForManager(ctx, msgID); err != nil {
				return fmt.Errorf("mark visible for manager: %v", err)
			}
			if _, err := s.outBox.Put(ctx, clientmessagesentjob.Name, v.MessageID, time.Now()); err != nil {
				return fmt.Errorf("put job %s: %v", clientmessagesentjob.Name, err)
			}
			return nil
		})
	case statusSuspicious:
		return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
			if err := s.msgRepo.BlockMessage(ctx, msgID); err != nil {
				return fmt.Errorf("block message: %v", err)
			}
			if _, err := s.outBox.Put(ctx, clientmessageblockedjob.Name, v.MessageID, time.Now()); err != nil {
				return fmt.Errorf("put job %s: %v", clientmessageblockedjob.Name, err)
			}
			return nil
		})
	default:
		return ErrUnknownStatus
	}
}

func (s *Service) decodeMsg(msg []byte) (verdict, types.MessageID, error) {
	var v verdict
	data := msg
	if s.publicKey != nil {
		parts := strings.Split(string(msg), ".")
		if len(parts) != 3 {
			return verdict{}, types.MessageIDNil, fmt.Errorf("%w: %d parts", ErrProcessingMessageWithKey, len(parts))
		}
		var err error
		if data, err = base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
			return verdict{}, types.MessageIDNil, fmt.Errorf("%w: decode body: %v", ErrProcessingMessageWithKey, err)
		}
		if err = (&jwt.SigningMethodRSA{Name: "RS256", Hash: crypto.SHA256}).
			Verify(strings.Join(parts[0:2], "."), parts[2], s.publicKey); err != nil {
			return verdict{}, types.MessageIDNil, fmt.Errorf("%w: verify signature: %v", ErrProcessingMessageWithKey, err)
		}
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return verdict{}, types.MessageIDNil, fmt.Errorf("unmarshal verdict: %v", err)
	}
	msgID, err := types.Parse[types.MessageID](v.MessageID)
	if err != nil {
		return verdict{}, types.MessageIDNil, fmt.Errorf("parse message id: %v", err)
	}
	return v, msgID, nil
}
