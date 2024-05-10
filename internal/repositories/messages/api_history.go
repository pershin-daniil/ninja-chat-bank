package messagesrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store/chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

const (
	minPageSize = 10
	maxPageSize = 100
)

var (
	ErrInvalidPageSize = errors.New("invalid page size")
	ErrInvalidCursor   = errors.New("invalid cursor")
)

type Cursor struct {
	LastCreatedAt time.Time
	PageSize      int
}

func (c Cursor) isValid() bool {
	if !isValidPageSize(c.PageSize) || c.LastCreatedAt.IsZero() {
		return false
	}

	return true
}

// GetClientChatMessages returns Nth page of messages in the chat for client side.
func (r *Repo) GetClientChatMessages(
	ctx context.Context,
	clientID types.UserID,
	pageSize int,
	cursor *Cursor,
) ([]Message, *Cursor, error) {
	query := r.db.Chat(ctx).Query().
		Where(chat.ClientIDEQ(clientID)).QueryMessages().
		Where(message.IsVisibleForClient(true))

	if cursor != nil {
		if !cursor.isValid() {
			return nil, nil, ErrInvalidCursor
		}

		query = query.Where(message.CreatedAtLT(cursor.LastCreatedAt))
		pageSize = cursor.PageSize
	}

	if !isValidPageSize(pageSize) {
		return nil, nil, ErrInvalidPageSize
	}

	messagesFromStore, err := query.Limit(pageSize + 1).Order(message.ByCreatedAt(sql.OrderDesc())).All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute the query: %v", err)
	}

	messagesCount := len(messagesFromStore)
	if messagesCount > pageSize {
		messagesCount = pageSize
	}

	messages := make([]Message, messagesCount)

	for i := 0; i < messagesCount; i++ {
		messages[i] = adaptStoreMessage(messagesFromStore[i])
	}

	cursor = nil
	if len(messagesFromStore) > len(messages) {
		cursor = &Cursor{
			LastCreatedAt: messages[messagesCount-1].CreatedAt,
			PageSize:      pageSize,
		}
	}

	return messages, cursor, nil
}

func isValidPageSize(p int) bool {
	if p >= minPageSize && p <= maxPageSize {
		return true
	}
	return false
}
