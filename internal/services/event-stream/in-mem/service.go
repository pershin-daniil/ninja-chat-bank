package inmemeventstream

import (
	"context"
	"fmt"
	"sync"
	"time"

	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type Service struct {
	subs   map[types.UserID][]*subscription
	mu     sync.RWMutex
	closed bool
}

type subscription struct {
	eventCh chan eventstream.Event
	closed  bool
	mu      sync.Mutex
}

func newSubscription(done <-chan struct{}) *subscription {
	sub := &subscription{eventCh: make(chan eventstream.Event)}
	go func() {
		<-done
		sub.mu.Lock()
		defer sub.mu.Unlock()
		if !sub.closed {
			sub.closed = true
			close(sub.eventCh)
		}
	}()

	return sub
}

func (sub *subscription) sendEvent(event eventstream.Event) {
	time.Sleep(10 * time.Millisecond)

	sub.mu.Lock()
	defer sub.mu.Unlock()

	if !sub.closed {
		sub.eventCh <- event
	}
}

func New() *Service {
	return &Service{
		subs: make(map[types.UserID][]*subscription),
	}
}

func (s *Service) Subscribe(ctx context.Context, userID types.UserID) (<-chan eventstream.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub := newSubscription(ctx.Done())
	s.subs[userID] = append(s.subs[userID], sub)

	return sub.eventCh, nil
}

func (s *Service) Publish(_ context.Context, userID types.UserID, event eventstream.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %v", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil
	}

	for _, sub := range s.subs[userID] {
		sub.sendEvent(event)
	}

	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true

	for _, subs := range s.subs {
		for _, sub := range subs {
			func() {
				sub.mu.Lock()
				defer sub.mu.Unlock()
				if !sub.closed {
					sub.closed = true
					close(sub.eventCh)
				}
			}()
		}
	}

	return nil
}
