package jobsrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/job"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

var ErrNoJobs = errors.New("no jobs found")

type Job struct {
	ID       types.JobID
	Name     string
	Payload  string
	Attempts int
}

func (r *Repo) FindAndReserveJob(ctx context.Context, until time.Time) (Job, error) {
	var j Job

	err := r.db.RunInTx(ctx, func(ctx context.Context) error {
		foundJob, err := r.db.Job(ctx).Query().Where(job.And(
			job.AvailableAtLT(time.Now()),
			job.ReservedUntilLT(time.Now()),
		)).Order(job.ByReservedUntil(entsql.OrderNullsFirst())).ForUpdate().First(ctx)

		switch {
		case store.IsNotFound(err):
			return fmt.Errorf("%w: %v", ErrNoJobs, err)
		case err != nil:
			return fmt.Errorf("failed to get job: %v", err)
		}

		foundJob, err = r.db.Job(ctx).UpdateOne(foundJob).SetReservedUntil(until).AddAttempts(1).Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update job: %v", err)
		}

		j = Job{
			ID:       foundJob.ID,
			Name:     foundJob.Name,
			Payload:  foundJob.Payload,
			Attempts: foundJob.Attempts,
		}

		return nil
	})
	if err != nil {
		return Job{}, fmt.Errorf("failed to run in tx: %w", err)
	}

	return j, nil
}

func (r *Repo) CreateJob(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error) {
	j, err := r.db.Job(ctx).Create().
		SetID(types.NewJobID()).
		SetName(name).
		SetPayload(payload).
		SetAvailableAt(availableAt).
		Save(ctx)
	if err != nil {
		return types.JobIDNil, fmt.Errorf("failed to create a job: %v", err)
	}

	return j.ID, nil
}

func (r *Repo) CreateFailedJob(ctx context.Context, name, payload, reason string) error {
	_, err := r.db.FailedJob(ctx).Create().
		SetName(name).
		SetPayload(payload).
		SetReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create a failed job: %v", err)
	}

	return nil
}

func (r *Repo) DeleteJob(ctx context.Context, jobID types.JobID) (err error) {
	if err = r.db.Job(ctx).DeleteOneID(jobID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete job: %v", err)
	}

	return nil
}
