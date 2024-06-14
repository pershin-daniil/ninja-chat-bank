package managerscheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	managerpool "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-pool"
	managerassignedtoproblemjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/manager-assigned-to-problem"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

const (
	serviceName                = "manager-scheduler"
	managerAssignedMsgTemplate = "manager %s will answer you"
)

type messagesRepo interface {
	CreateServiceClientVisible(
		ctx context.Context,
		reqID types.RequestID,
		problemID types.ProblemID,
		chatID types.ChatID,
		msgBody string,
	) (*messagesrepo.Message, error)
	GetInitialMessageByProblemID(ctx context.Context, problemID types.ProblemID) (*messagesrepo.Message, error)
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type problemsRepo interface {
	ProblemsWithoutManager(ctx context.Context, limit int) ([]types.ProblemID, error)
	AssignManagerToProblem(ctx context.Context, managerID types.UserID, problemID types.ProblemID) (*problemsrepo.Problem, error)
	ProblemInitialMessageRequestID(ctx context.Context, problemID types.ProblemID) (types.RequestID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	idleDuration time.Duration    `option:"mandatory" validate:"min=100ms,max=1m"`
	mngrPool     managerpool.Pool `option:"mandatory" validate:"required"`
	msgRepo      messagesRepo     `option:"mandatory" validate:"required"`
	outbox       outboxService    `option:"mandatory" validate:"required"`
	problemsRepo problemsRepo     `option:"mandatory" validate:"required"`
	txtor        transactor       `option:"mandatory" validate:"required"`
	log          *zap.Logger      `option:"mandatory" validate:"required"`
}

type Service struct {
	Options
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}
	opts.log = opts.log.Named(serviceName)
	return &Service{Options: opts}, nil
}

func (s *Service) Run(ctx context.Context) error {
	for {
		if s.mngrPool.Size() > 0 {
			if err := s.assignManagers(ctx); err != nil {
				return fmt.Errorf("assign managers: %v", err)
			}
		}

		sleep := time.NewTimer(s.idleDuration)
		select {
		case <-sleep.C:
		case <-ctx.Done():
			sleep.Stop()
			return nil
		}
	}
}

func (s *Service) assignManagers(ctx context.Context) error {
	problemIDs, err := s.problemsRepo.ProblemsWithoutManager(ctx, s.mngrPool.Size())
	if err != nil {
		return fmt.Errorf("get problems without manager: %v", err)
	}

	var managerID types.UserID
	for _, problemID := range problemIDs {
		managerID, err = s.mngrPool.Get(ctx)
		if err != nil {
			if errors.Is(err, managerpool.ErrNoAvailableManagers) {
				return nil
			}
			return fmt.Errorf("get manager from pool: %v", err)
		}

		problem, err := s.problemsRepo.AssignManagerToProblem(ctx, managerID, problemID)
		if err != nil {
			s.log.Warn("assign manager to problem error",
				zap.Error(err),
				zap.Stringer("manager_id", managerID),
				zap.Stringer("problem_id", problemID),
			)
			return fmt.Errorf("assign manager to problem: %v", err)
		}

		err = s.notifyClientAboutAssignment(ctx, problem, managerID)
		if err != nil {
			s.log.Warn("notify client error", zap.Error(err), zap.Stringer("problem_id", problem.ID))
			return fmt.Errorf("notify client: %v", err)
		}
	}
	return nil
}

func (s *Service) notifyClientAboutAssignment(ctx context.Context, problem *problemsrepo.Problem, managerID types.UserID) error {
	// requestID, err := s.problemsRepo.ProblemInitialMessageRequestID(ctx, problem.ID)
	initialMessage, err := s.msgRepo.GetInitialMessageByProblemID(ctx, problem.ID)
	if err != nil {
		return fmt.Errorf("find problem request id: %v", err)
	}

	return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
		text := managerAssignedMessageText(managerID)
		msg, err := s.msgRepo.CreateServiceClientVisible(ctx, types.NewRequestID(), problem.ID, problem.ChatID, text)
		if err != nil {
			return fmt.Errorf("create service message: %v", err)
		}

		payload, err := managerassignedtoproblemjob.Marshal(managerassignedtoproblemjob.Payload{
			MessageID: msg.ID,
			ManagerID: managerID,
			ClientID:  initialMessage.AuthorID,
		})
		if err != nil {
			return fmt.Errorf("marshal job %q payload: %v", managerassignedtoproblemjob.Name, err)
		}
		_, err = s.outbox.Put(ctx, managerassignedtoproblemjob.Name, payload, time.Now())
		if err != nil {
			return fmt.Errorf("put job %q: %v", managerassignedtoproblemjob.Name, err)
		}

		return nil
	})
}

func managerAssignedMessageText(managerID types.UserID) string {
	return fmt.Sprintf(managerAssignedMsgTemplate, managerID)
}
