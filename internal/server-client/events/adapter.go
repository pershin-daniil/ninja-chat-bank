package clientevents

import (
	"fmt"

	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	websocketstream "github.com/pershin-daniil/ninja-chat-bank/internal/websocket-stream"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

var _ websocketstream.EventAdapter = Adapter{}

type Adapter struct{}

func (Adapter) Adapt(ev eventstream.Event) (any, error) {
	if err := ev.Validate(); err != nil {
		return nil, fmt.Errorf("event validate: %v", err)
	}

	switch t := ev.(type) {
	case *eventstream.NewMessageEvent:
		event := Event{}
		err := event.FromNewMessageEvent(
			NewMessageEvent{
				AuthorID:  pointer.PtrWithZeroAsNil(t.UserID),
				Body:      t.MessageBody,
				CreatedAt: &t.Time,
				ID:        t.ID,
				EventType: EventTypeNewMessageEvent,
				IsService: &t.IsService,
				MessageID: t.MessageID,
				RequestID: t.RequestID,
			})
		if err != nil {
			return nil, fmt.Errorf("from new message event: %v", err)
		}
		return event, nil

	case *eventstream.MessageSentEvent:
		event := Event{}
		err := event.FromMessageSentEvent(MessageSentEvent{
			ID:        t.ID,
			EventType: EventTypeMessageSentEvent,
			MessageID: t.MessageID,
			RequestID: t.RequestID,
		})
		if err != nil {
			return nil, fmt.Errorf("from message sent event: %v", err)
		}
		return event, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", t)
	}
}
