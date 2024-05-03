package sendmessage

import (
	"context"
	"errors"
	"fmt"
	"time"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	sendclientmessagejob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/send-client-message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=sendmessagemocks

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrChatNotCreated    = errors.New("chat not created")
	ErrProblemNotCreated = errors.New("problem not created")
)

type chatsRepository interface {
	CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error)
}

type outboxService interface {
	Put(ctx context.Context, name string, payload string, availableAt time.Time) (types.JobID, error)
}

type messagesRepository interface {
	GetMessageByRequestID(ctx context.Context, reqID types.RequestID) (*messagesrepo.Message, error)
	CreateClientVisible(
		ctx context.Context,
		reqID types.RequestID,
		problemID types.ProblemID,
		chatID types.ChatID,
		authorID types.UserID,
		msgBody string,
	) (*messagesrepo.Message, error)
}

type problemsRepository interface {
	CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=sendmessage.gen.go -from-struct=Options
type Options struct {
	chatRepo      chatsRepository    `option:"mandatory" validate:"required"`
	msgRepo       messagesRepository `option:"mandatory" validate:"required"`
	outboxService outboxService      `option:"mandatory" validate:"required"`
	problemRepo   problemsRepository `option:"mandatory" validate:"required"`
	tx            transactor         `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	return UseCase{Options: opts}, opts.Validate()
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	var msg *messagesrepo.Message

	err := u.tx.RunInTx(ctx, func(ctx context.Context) (err error) {
		msg, err = u.msgRepo.GetMessageByRequestID(ctx, req.ID)
		switch {
		case nil == err:
			return nil
		case !errors.Is(err, messagesrepo.ErrMsgNotFound):
			return fmt.Errorf("failed to get message by req id: %v", err)
		}

		chatID, err := u.chatRepo.CreateIfNotExists(ctx, req.ClientID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrChatNotCreated, err)
		}

		problemID, err := u.problemRepo.CreateIfNotExists(ctx, chatID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProblemNotCreated, err)
		}

		msg, err = u.msgRepo.CreateClientVisible(ctx, req.ID, problemID, chatID, req.ClientID, req.MessageBody)
		if err != nil {
			return fmt.Errorf("failed to create msg: %v", err)
		}

		if _, err = u.outboxService.Put(ctx, sendclientmessagejob.Name, msg.ID.String(), time.Now()); err != nil {
			return fmt.Errorf("failed to put message: %v", err)
		}

		return nil
	})
	if err != nil {
		return Response{}, fmt.Errorf("failed to run in tx: %w", err)
	}

	return Response{
		MessageID: msg.ID,
		AuthorID:  msg.AuthorID,
		CreatedAt: msg.CreatedAt,
	}, nil
}
