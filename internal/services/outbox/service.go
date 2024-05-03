package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	jobsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/jobs"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

const (
	serviceName = "outbox"

	reasonJobNotRegistered       = "not_registered"
	reasonMaxJobAttemptsExceeded = "attempts_exceeded"
)

var ErrJobAlreadyExists = errors.New("job already exists")

type jobsRepository interface {
	CreateJob(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
	CreateFailedJob(ctx context.Context, name, payload, reason string) error
	FindAndReserveJob(ctx context.Context, until time.Time) (jobsrepo.Job, error)
	DeleteJob(ctx context.Context, jobID types.JobID) error
}

type transactor interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	workers    int            `option:"mandatory" validate:"min=1,max=32"`
	idleTime   time.Duration  `option:"mandatory" validate:"min=100ms,max=10s"`
	reserveFor time.Duration  `option:"mandatory" validate:"min=1s,max=10m"`
	jobsRepo   jobsRepository `option:"mandatory"`
	db         transactor     `option:"mandatory"`
}

type Service struct {
	workers    int
	idleTime   time.Duration
	reserveFor time.Duration
	jobsRepo   jobsRepository
	db         transactor
	registry   map[string]Job
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate options outbox: %v", err)
	}

	return &Service{
		workers:    opts.workers,
		idleTime:   opts.idleTime,
		reserveFor: opts.reserveFor,
		jobsRepo:   opts.jobsRepo,
		db:         opts.db,
		registry:   make(map[string]Job),
	}, nil
}

func (s *Service) RegisterJob(job Job) error {
	if _, ok := s.registry[job.Name()]; ok {
		return ErrJobAlreadyExists
	}

	s.registry[job.Name()] = job

	return nil
}

func (s *Service) MustRegisterJob(job Job) {
	if err := s.RegisterJob(job); err != nil {
		panic(err)
	}
}

func (s *Service) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	for i := 0; i < s.workers; i++ {
		lg := zap.L().With(zap.String("service", serviceName), zap.Int("worker_id", i+1))

		eg.Go(func() error {
			for {
				if err := s.executeAvailableJobs(ctx, lg); err != nil {
					if ctx.Err() != nil {
						return err
					}

					lg.Warn("execute error", zap.Error(err))
					return err
				}

				select {
				case <-ctx.Done():
					return nil
				case <-time.After(s.idleTime):
				}
			}
		})
	}

	return eg.Wait()
}

func (s *Service) executeAvailableJobs(ctx context.Context, lg *zap.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := s.execute(ctx, lg); err != nil {
			if errors.Is(err, jobsrepo.ErrNoJobs) {
				lg.Debug("no jobs", zap.Error(err))
				return nil
			}

			return err
		}
	}
}

func (s *Service) execute(ctx context.Context, lg *zap.Logger) error {
	task, err := s.jobsRepo.FindAndReserveJob(ctx, time.Now().Add(s.reserveFor))
	if err != nil {
		return fmt.Errorf("failed to find and reserve job: %w", err)
	}

	l := lg.With(
		zap.String("job_name", task.Name),
		zap.Stringer("job_id", task.ID),
		zap.String("payload", task.Payload),
		zap.Int("attempts", task.Attempts))
	l.Info("executing task")

	job, ok := s.registry[task.Name]
	if !ok {
		return s.dlq(ctx, task, reasonJobNotRegistered)
	}

	func() {
		c, cancel := context.WithTimeout(ctx, job.ExecutionTimeout())
		defer cancel()

		err = job.Handle(c, task.Payload)
	}()

	if err != nil {
		l.Warn("failed to handle job", zap.Error(err))

		if task.Attempts >= job.MaxAttempts() {
			return s.dlq(ctx, task, reasonMaxJobAttemptsExceeded)
		}

		return nil
	}

	// Delete job with context.Background() to prevent handling a job when ctx is already closed.
	if err = s.jobsRepo.DeleteJob(ctx, task.ID); err != nil {
		l.Warn("failed to delete job", zap.Error(err))
	}

	return nil
}

func (s *Service) dlq(ctx context.Context, task jobsrepo.Job, reason string) error {
	return s.db.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.jobsRepo.CreateFailedJob(ctx, task.Name, task.Payload, reason); err != nil {
			return fmt.Errorf("failed to create failed job: %v", err)
		}

		if err := s.jobsRepo.DeleteJob(ctx, task.ID); err != nil {
			return fmt.Errorf("failed to delete job in dlq: %v", err)
		}

		return nil
	})
}
