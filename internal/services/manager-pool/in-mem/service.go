package inmemmanagerpool

import (
	"context"
	"errors"
	managerpool "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-pool"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"sync"
)

const (
	serviceName = "manager-pool"
	managersMax = 1000
)

var ErrManagerCapacityExceeded = errors.New("too many managers")

type Service struct {
	queue    []types.UserID
	managers map[types.UserID]struct{}
	mu       sync.RWMutex
}

func New() *Service {
	return &Service{
		managers: make(map[types.UserID]struct{}),
	}
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queue = s.queue[:0]
	s.managers = make(map[types.UserID]struct{})

	return nil
}

func (s *Service) Get(_ context.Context) (types.UserID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return types.UserIDNil, managerpool.ErrNoAvailableManagers
	}

	manager := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.managers, manager)

	return manager, nil
}

func (s *Service) Put(ctx context.Context, managerID types.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.managers) >= managersMax {
		return ErrManagerCapacityExceeded
	}

	if _, ok := s.managers[managerID]; ok {
		return nil
	}

	s.managers[managerID] = struct{}{}
	s.queue = append(s.queue, managerID)

	return nil
}

func (s *Service) Contains(_ context.Context, managerID types.UserID) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.managers[managerID]

	return ok, nil
}

func (s *Service) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.managers)
}
