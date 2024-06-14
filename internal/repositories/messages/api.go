package messagesrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/problem"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
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

func (r *Repo) GetInitialMessageByProblemID(ctx context.Context, problemID types.ProblemID) (*Message, error) {
	msg, err := r.db.Message(ctx).
		Query().
		Where(
			message.HasChatWith(chat.HasProblemsWith(problem.ID(problemID))),
			message.IsVisibleForManager(true),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("find message: %v", err)
	}

	return pointer.Ptr(adaptStoreMessage(msg)), nil
}

func (r *Repo) CreateServiceClientVisible(
	ctx context.Context,
	reqID types.RequestID,
	problemID types.ProblemID,
	chatID types.ChatID,
	msgBody string,
) (*Message, error) {
	msg, err := r.db.Message(ctx).
		Create().
		SetInitialRequestID(reqID).
		SetProblemID(problemID).
		SetChatID(chatID).
		SetBody(msgBody).
		SetIsVisibleForClient(true).
		SetIsVisibleForManager(false).
		SetIsService(true).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create message: %v", err)
	}
	return pointer.Ptr(adaptStoreMessage(msg)), nil
}
