package clientevents

import (
	"errors"
	"fmt"

	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	websocketstream "github.com/pershin-daniil/ninja-chat-bank/internal/websocket-stream"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

var ErrUnexpectedEventType = errors.New("unexpected event type")

var _ websocketstream.EventAdapter = Adapter{}

type Adapter struct{}

func (Adapter) Adapt(ev eventstream.Event) (any, error) {
	switch e := ev.(type) {
	case *eventstream.NewMessageEvent:
		event := Event{}

		err := event.FromNewMessageEvent(
			NewMessageEvent{
				AuthorId:  pointer.PtrWithZeroAsNil(e.UserID),
				Body:      e.MessageBody,
				CreatedAt: e.CreatedAt,
				EventId:   e.EventID,
				IsService: e.IsService,
				MessageId: e.MessageID,
				RequestId: e.RequestID,
			})
		if err != nil {
			return nil, fmt.Errorf("from new message event: %v", err)
		}

		return event, nil
	case *eventstream.MessageSentEvent:
		event := Event{}

		err := event.FromMessageSentEvent(MessageSentEvent{
			EventId:   e.EventID,
			MessageId: e.MessageID,
			RequestId: e.RequestID,
		})
		if err != nil {
			return nil, fmt.Errorf("from new message event: %v", err)
		}

		return event, nil
	case *eventstream.MessageBlockEvent:
		event := Event{}

		err := event.FromMessageBlockedEvent(MessageBlockedEvent{
			EventId:   e.EventID,
			MessageId: e.MessageID,
			RequestId: e.RequestID,
		})
		if err != nil {
			return nil, fmt.Errorf("from new message event: %v", err)
		}

		return event, nil
	}
	return nil, ErrUnexpectedEventType
}
