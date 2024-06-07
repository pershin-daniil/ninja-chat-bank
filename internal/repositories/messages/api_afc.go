package messagesrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func (r *Repo) MarkAsVisibleForManager(ctx context.Context, msgID types.MessageID) error {
	err := r.db.Message(ctx).
		UpdateOneID(msgID).
		SetIsVisibleForManager(true).
		SetCheckedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark as visible for manager: %v", err)
	}

	return nil
}

func (r *Repo) BlockMessage(ctx context.Context, msgID types.MessageID) error {
	err := r.db.Message(ctx).
		UpdateOneID(msgID).
		SetIsBlocked(true).
		SetCheckedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("block message: %v", err)
	}

	return nil
}
