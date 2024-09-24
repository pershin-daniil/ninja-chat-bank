package closechat

import (
	"context"
	"errors"
	"fmt"
	"time"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	closechatjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/close-chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -typed -package=closechatsmocks

const ProblemResolvedMsgText = "Your question has been marked as resolved.\nThank you for being with us!"

type problemsRepository interface {
	ResolveAssignedProblem(ctx context.Context, chatID types.ChatID, managerID types.UserID) (*problemsrepo.Problem, error)
}

type messagesRepository interface {
	CreateServiceClientVisible(
		ctx context.Context,
		reqID types.RequestID,
		problemID types.ProblemID,
		chatID types.ChatID,
		msgBody string,
	) (*messagesrepo.Message, error)
}

type chatsRepository interface {
	GetClientIDByChatID(ctx context.Context, chatID types.ChatID) (types.UserID, error)
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	problemsRepo problemsRepository `option:"mandatory" validate:"required"`
	messagesRepo messagesRepository `option:"mandatory" validate:"required"`
	chatsRepo    chatsRepository    `option:"mandatory" validate:"required"`
	outbox       outboxService      `option:"mandatory" validate:"required"`
	transactor   transactor         `option:"mandatory" validate:"required"`
}

// New creates a new UseCase instance based on the provided options.
// It validates the options and returns an initialized UseCase instance
// if the validation succeeds. Otherwise, it returns an error.
func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("validate options: %v", err)
	}

	return UseCase{Options: opts}, nil
}

type UseCase struct {
	Options
}

func (u UseCase) Handle(ctx context.Context, req Request) error {
	if err := validator.Validator.Struct(req); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	clientID, err := u.chatsRepo.GetClientIDByChatID(ctx, req.ChatID)
	if err != nil {
		return fmt.Errorf("get client id by chat id: %v", err)
	}

	tx := func(ctx context.Context) error {
		problem, err := u.problemsRepo.ResolveAssignedProblem(ctx, req.ChatID, req.ManagerID)
		if err != nil {
			if errors.Is(err, problemsrepo.ErrProblemNotFound) {
				return ErrNoActiveProblemInChat
			}
			return fmt.Errorf("resolve open problem: %v", err)
		}

		msg, err := u.messagesRepo.CreateServiceClientVisible(ctx, req.RequestID, problem.ID, req.ChatID, ProblemResolvedMsgText)
		if err != nil {
			return fmt.Errorf("create client message: %v", err)
		}

		payload, err := closechatjob.Marshal(closechatjob.Payload{
			RequestID: req.RequestID,
			ManagerID: req.ManagerID,
			MessageID: msg.ID,
			ClientID:  clientID,
		})
		if err != nil {
			return fmt.Errorf("marshal %q job payload: %v", closechatjob.Name, err)
		}
		_, err = u.outbox.Put(ctx, closechatjob.Name, payload, time.Now())
		if err != nil {
			return fmt.Errorf("put %q job: %v", closechatjob.Name, err)
		}

		return nil
	}

	return u.transactor.RunInTx(ctx, tx)
}
