package managerevents

import (
	"fmt"

	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	websocketstream "github.com/pershin-daniil/ninja-chat-bank/internal/websocket-stream"
)

var _ websocketstream.EventAdapter = Adapter{}

type Adapter struct{}

func (Adapter) Adapt(ev eventstream.Event) (any, error) {
	if err := ev.Validate(); err != nil {
		return nil, fmt.Errorf("event validate: %v", err)
	}
	switch t := ev.(type) {
	case *eventstream.NewChatEvent:
		return NewChatEvent{
			ID:                  t.EventID,
			ChatID:              t.ChatID,
			ClientID:            t.ClientID,
			EventType:           string(EventTypeNewChatEvent),
			RequestID:           t.RequestID,
			CanTakeMoreProblems: t.CanTakeMoreProblems,
		}, nil
	case *eventstream.NewMessageEvent:
		return NewMessageEvent{
			ClientID:  t.AuthorID,
			Body:      t.MessageBody,
			ChatID:    t.ChatID,
			CreatedAt: t.CreatedAt,
			ID:        t.EventID,
			EventType: string(EventTypeNewMessageEvent),
			MessageID: t.MessageID,
			RequestID: t.RequestID,
		}, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", t)
	}
}
