package messagesrepo

import (
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type Message struct {
	ID                  types.MessageID
	ChatID              types.ChatID
	AuthorID            types.UserID
	RequestID           types.RequestID
	ProblemID           types.ProblemID
	Body                string
	CreatedAt           time.Time
	IsVisibleForClient  bool
	IsVisibleForManager bool
	IsBlocked           bool
	IsService           bool
	InitialRequestID    types.RequestID
}

func adaptStoreMessage(m *store.Message) Message {
	return Message{
		ID:                  m.ID,
		ChatID:              m.ChatID,
		AuthorID:            m.AuthorID,
		RequestID:           m.InitialRequestID,
		ProblemID:           m.ProblemID,
		Body:                m.Body,
		CreatedAt:           m.CreatedAt,
		IsVisibleForClient:  m.IsVisibleForClient,
		IsVisibleForManager: m.IsVisibleForManager,
		IsBlocked:           m.IsBlocked,
		IsService:           m.IsService,
		InitialRequestID:    m.InitialRequestID,
	}
}
