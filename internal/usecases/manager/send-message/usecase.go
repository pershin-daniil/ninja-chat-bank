package sendmessage

import (
	"context"
	"fmt"
	"time"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/payload/simpleid"
	sendmanagermessagejob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/send-manager-message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -typed -package=sendmessagemocks

type messagesRepository interface {
	CreateFullVisible(
		ctx context.Context,
		reqID types.RequestID,
		problemID types.ProblemID,
		chatID types.ChatID,
		authorID types.UserID,
		msgBody string,
	) (*messagesrepo.Message, error)
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type problemsRepository interface {
	GetAssignedProblemID(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	messagesRepository messagesRepository `option:"mandatory" validate:"required"`
	outboxService      outboxService      `option:"mandatory" validate:"required"`
	problemsRepository problemsRepository `option:"mandatory" validate:"required"`
	transactor         transactor         `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("validate options: %v", err)
	}
	return UseCase{Options: opts}, nil
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := validator.Validator.Struct(req); err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	problemID, err := u.problemsRepository.GetAssignedProblemID(ctx, req.ManagerID, req.ChatID)
	if err != nil {
		return Response{}, fmt.Errorf("get assigned problem: %v", err)
	}

	var msg *messagesrepo.Message
	tx := func(ctx context.Context) error {
		msg, err = u.messagesRepository.CreateFullVisible(ctx, req.ID, problemID, req.ChatID, req.ManagerID, req.MessageBody)
		if err != nil {
			return fmt.Errorf("create full visible message: %v", err)
		}

		var payload string
		if payload, err = simpleid.Marshal[types.MessageID](msg.ID); err != nil {
			return fmt.Errorf("marshal %q job payload: %v", sendmanagermessagejob.Name, err)
		}
		_, err = u.outboxService.Put(ctx, sendmanagermessagejob.Name, payload, time.Now())
		if err != nil {
			return fmt.Errorf("send %q job: %v", sendmanagermessagejob.Name, err)
		}

		return nil
	}

	if err := u.transactor.RunInTx(ctx, tx); err != nil {
		return Response{}, fmt.Errorf("transaction error: %v", err)
	}

	return Response{MessageID: msg.ID, CreatedAt: msg.CreatedAt}, nil
}
