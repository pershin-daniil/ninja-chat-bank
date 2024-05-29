package eventstream

import (
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

type Event interface {
	eventMarker()
	Validate() error
}

type event struct{}         //
func (*event) eventMarker() {}

// MessageSentEvent indicates that the message was checked by AFC
// and was sent to the manager. Two gray ticks.
type MessageSentEvent struct {
	event
	ID        types.EventID   `validate:"required"`
	MessageID types.MessageID `validate:"required"`
	RequestID types.RequestID `validate:"required"`
}

func NewMessageSentEvent(id types.EventID, requestID types.RequestID, messageID types.MessageID) *MessageSentEvent {
	return &MessageSentEvent{
		ID:        id,
		RequestID: requestID,
		MessageID: messageID,
	}
}

func (e MessageSentEvent) Validate() error {
	return validator.Validator.Struct(e)
}

// NewMessageEvent is a signal about the appearance of a new message in the chat.
type NewMessageEvent struct {
	event
	ID          types.EventID   `validate:"required"`
	RequestID   types.RequestID `validate:"required"`
	ChatID      types.ChatID    `validate:"required"`
	MessageID   types.MessageID `validate:"required"`
	UserID      types.UserID    `validate:"required_if=IsService false"`
	Time        time.Time       `validate:"required"`
	MessageBody string          `validate:"required"`
	IsService   bool
}

func NewNewMessageEvent(
	eventID types.EventID,
	requestID types.RequestID,
	chatID types.ChatID,
	messageID types.MessageID,
	userID types.UserID,
	t time.Time,
	body string,
	isService bool,
) Event {
	return &NewMessageEvent{
		event:       event{},
		ID:          eventID,
		RequestID:   requestID,
		ChatID:      chatID,
		MessageID:   messageID,
		UserID:      userID,
		Time:        t,
		MessageBody: body,
		IsService:   isService,
	}
}

func (e NewMessageEvent) Validate() error {
	return validator.Validator.Struct(e)
}
