package chatsrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error) {
	chatID, err := r.db.Chat(ctx).Create().
		SetID(types.NewChatID()).
		SetCreatedAt(time.Now()).
		SetClientID(userID).
		OnConflictColumns("client_id").
		Ignore().
		ID(ctx)
	if err != nil {
		return types.ChatIDNil, fmt.Errorf("failed to create chat: %v", err)
	}

	return chatID, nil
}
