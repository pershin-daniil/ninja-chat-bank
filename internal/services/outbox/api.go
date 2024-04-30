package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func (s *Service) Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error) {
	id, err := s.jobsRepo.CreateJob(ctx, name, payload, availableAt)
	if err != nil {
		return types.JobIDNil, fmt.Errorf("failed to put a job: %v", err)
	}

	return id, nil
}
