package chatsrepo

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store/chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error) {
	chatID, err := r.db.Chat(ctx).Create().
		SetClientID(userID).
		OnConflict(
			sql.ConflictColumns(chat.FieldClientID),
			sql.ResolveWith(func(set *sql.UpdateSet) {
				set.SetIgnore(chat.FieldClientID)
			}),
		).
		ID(ctx)
	if err != nil {
		return types.ChatIDNil, fmt.Errorf("failed to create chat: %v", err)
	}

	return chatID, nil
}
