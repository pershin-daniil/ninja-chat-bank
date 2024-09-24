package sendmanagermessagejob

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	msgproducer "github.com/pershin-daniil/ninja-chat-bank/internal/services/msg-producer"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/payload/simpleid"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/job_mock.gen.go -typed -package=sendmanagermessagejobmocks

const Name = "send-manager-message"

type messageProducer interface {
	ProduceMessage(ctx context.Context, message msgproducer.Message) error
}

type messagesRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

type chatsRepository interface {
	GetClientIDByChatID(ctx context.Context, chatID types.ChatID) (types.UserID, error)
}

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	producer        messageProducer    `option:"mandatory" validate:"required"`
	messagesRepo    messagesRepository `option:"mandatory" validate:"required"`
	chatsRepository chatsRepository    `option:"mandatory" validate:"required"`
	evStream        eventStream        `option:"mandatory" validate:"required"`
}

func Must(opts Options) *Job {
	j, err := New(opts)
	if err != nil {
		panic(err)
	}
	return j
}

func New(opts Options) (*Job, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}
	return &Job{
		Options: opts,
		logger:  zap.L().Named(Name),
	}, nil
}

type Job struct {
	Options
	outbox.DefaultJob
	logger *zap.Logger
}

func (j *Job) Name() string {
	return Name
}

func (j *Job) Handle(ctx context.Context, payload string) error {
	j.logger.Info("start processing", zap.String("payload", payload))

	msgID, err := simpleid.Unmarshal[types.MessageID](payload)
	if err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	// Manager message
	msg, err := j.messagesRepo.GetMessageByID(ctx, msgID)
	if err != nil {
		return fmt.Errorf("get message: %v", err)
	}

	err = j.producer.ProduceMessage(ctx, msgproducer.Message{
		ID:         msg.ID,
		ChatID:     msg.ChatID,
		Body:       msg.Body,
		FromClient: false,
	})
	if err != nil {
		j.logger.Error("produce message", zap.Error(err), zap.String("id", msgID.String()))
		return fmt.Errorf("produce message: %v", err)
	}

	clientID, err := j.chatsRepository.GetClientIDByChatID(ctx, msg.ChatID)
	if err != nil {
		return fmt.Errorf("get client id for chat: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Publish NewMessageEvent to manager
	eg.Go(func() error {
		if err := j.publishNewMessageEvent(ctx, msg, msg.AuthorID); err != nil {
			j.logger.Error("publish NewMessageEvent to manager error", zap.Error(err))
			return fmt.Errorf("publish NewMessageEvent to manager: %v", err)
		}

		return nil
	})

	// Publish NewMessageEvent to client
	eg.Go(func() error {
		if err := j.publishNewMessageEvent(ctx, msg, clientID); err != nil {
			j.logger.Error("publish NewMessageEvent to client error", zap.Error(err))
			return fmt.Errorf("publish NewMessageEvent to client: %v", err)
		}

		return nil
	})

	return eg.Wait()
}

func (j *Job) publishNewMessageEvent(ctx context.Context, msg *messagesrepo.Message, userID types.UserID) error {
	eventID := types.NewEventID()
	event := eventstream.NewNewMessageEvent(
		eventID,
		msg.InitialRequestID,
		msg.ChatID,
		msg.ID,
		msg.AuthorID,
		msg.CreatedAt,
		msg.Body,
		msg.IsService,
	)
	err := j.evStream.Publish(ctx, userID, event)
	if err != nil {
		j.logger.Error("publish new message event", zap.Error(err), zap.String("id", msg.ID.String()))
		return err
	}
	j.logger.Debug(
		"NewMessageEvent published",
		zap.Stringer("event_id", eventID),
		zap.Stringer("author_id", msg.AuthorID),
		zap.Stringer("msg_id", msg.ID),
	)

	j.logger.Debug("message produced", zap.String("id", msg.ID.String()))
	return nil
}
