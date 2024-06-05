package inmemeventstream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

var ErrEventStreamClosed = errors.New("event stream closed")

type Service struct {
	clients *clients

	closed bool

	mu sync.Mutex
	wg sync.WaitGroup
}

func New() *Service {
	return &Service{
		clients: newClients(),
		closed:  false,
	}
}

func (s *Service) Subscribe(ctx context.Context, userID types.UserID) (<-chan eventstream.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, ErrEventStreamClosed
	}

	client := s.clients.add(ctx, userID)

	return client.ch, nil
}

func (s *Service) Publish(_ context.Context, userID types.UserID, event eventstream.Event) error {
	s.wg.Add(1)
	defer s.wg.Done()

	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate event: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrEventStreamClosed
	}

	for _, c := range s.clients.get(userID) {
		select {
		case <-c.ctx.Done():
			s.clients.remove(c)
			continue
		default:
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-timer.C:
			s.clients.remove(c)
			continue
		case c.ch <- event:
		}
	}

	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	s.wg.Wait()
	return nil
}
