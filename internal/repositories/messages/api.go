package messagesrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

var ErrMsgNotFound = errors.New("message not found")

func (r *Repo) GetMessageByRequestID(ctx context.Context, reqID types.RequestID) (*Message, error) {
	msg, err := r.db.Message(ctx).Query().
		Unique(false).
		Where(message.InitialRequestID(reqID)).
		Only(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, fmt.Errorf("request id: %v: %w", reqID, ErrMsgNotFound)
		}
		return nil, fmt.Errorf("query message by request id: %v: %v", reqID, err)
	}

	m := adaptStoreMessage(msg)

	return &m, nil
}

func (r *Repo) GetMessageByID(ctx context.Context, id types.MessageID) (*Message, error) {
	msg, err := r.db.Message(ctx).Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("query message by id: %v", err)
	}

	m := adaptStoreMessage(msg)

	return &m, nil
}

// CreateClientVisible creates a message that is visible only to the client.
func (r *Repo) CreateClientVisible(
	ctx context.Context,
	reqID types.RequestID,
	problemID types.ProblemID,
	chatID types.ChatID,
	authorID types.UserID,
	msgBody string,
) (*Message, error) {
	msg, err := r.db.Message(ctx).Create().
		SetChatID(chatID).
		SetProblemID(problemID).
		SetAuthorID(authorID).
		SetIsVisibleForClient(true).
		SetIsVisibleForManager(false).
		SetBody(msgBody).
		SetInitialRequestID(reqID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create msg: %v", err)
	}

	m := adaptStoreMessage(msg)

	return &m, nil
}
