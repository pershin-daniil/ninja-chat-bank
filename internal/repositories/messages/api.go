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
	msg, err := r.db.Message(ctx).Query().Where(message.InitialRequestID(reqID)).Only(ctx)
	switch {
	case store.IsNotFound(err):
		return nil, fmt.Errorf("%w: %v", ErrMsgNotFound, err)
	case err != nil:
		return nil, fmt.Errorf("failed to get messages by reqID: %v", err)
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
		SetID(types.NewMessageID()).
		SetInitialRequestID(reqID).
		SetProblemID(problemID).
		SetChatID(chatID).
		SetAuthorID(authorID).
		SetBody(msgBody).
		SetIsVisibleForClient(true).
		SetIsVisibleForManager(false).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %v", err)
	}

	m := adaptStoreMessage(msg)

	return &m, nil
}
