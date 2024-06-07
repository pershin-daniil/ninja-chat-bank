package clientmessagesentjob

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/payload/simpleid"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/job_mock.gen.go -package=clientmessagesentjobmocks

const Name = "client-message-sent"

type eventStream interface {
	Publish(ctx context.Context, userID types.UserID, event eventstream.Event) error
}

type messageRepository interface {
	GetMessageByID(ctx context.Context, msgID types.MessageID) (*messagesrepo.Message, error)
}

//go:generate options-gen -out-filename=job_options.gen.go -from-struct=Options
type Options struct {
	eventStream eventStream       `option:"mandatory" validate:"required"`
	msgRepo     messageRepository `option:"mandatory" validate:"required"`
}

type Job struct {
	outbox.DefaultJob
	Options
	logger *zap.Logger
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
		return &Job{}, fmt.Errorf("validate options: %v", err)
	}
	return &Job{
		Options: opts,
		logger:  zap.L().Named("job." + Name),
	}, nil
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

	msg, err := j.msgRepo.GetMessageByID(ctx, msgID)
	if err != nil {
		return fmt.Errorf("get message: %v", err)
	}

	wg, ctx := errgroup.WithContext(ctx)

	// Send update to client.
	wg.Go(func() error {
		if err = j.eventStream.Publish(ctx, msg.AuthorID,
			eventstream.NewMessageSentEvent(
				types.NewEventID(),
				msg.InitialRequestID,
				msg.ID,
			),
		); err != nil {
			return fmt.Errorf("publish MessageSentEvent to client: %v", err)
		}
		return nil
	})

	return wg.Wait()
}
