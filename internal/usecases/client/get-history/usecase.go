package gethistory

import (
	"context"
	"errors"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/cursor"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=gethistorymocks

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidCursor  = errors.New("invalid cursor")
)

type messagesRepository interface {
	GetClientChatMessages(
		ctx context.Context,
		clientID types.UserID,
		pageSize int,
		cursor *messagesrepo.Cursor,
	) ([]messagesrepo.Message, *messagesrepo.Cursor, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	msgRepo messagesRepository `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("validate options gethistory: %v", err)
	}

	return UseCase{
		Options: opts,
	}, nil
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, fmt.Errorf("failed to validate: %w", err)
	}

	var cursorParam *messagesrepo.Cursor
	if req.Cursor != "" {
		var reqCursor messagesrepo.Cursor
		if err := cursor.Decode(req.Cursor, &reqCursor); err != nil {
			return Response{}, ErrInvalidCursor
		}

		cursorParam = &reqCursor
	}

	messages, respCursor, err := u.msgRepo.GetClientChatMessages(ctx, req.ClientID, req.PageSize, cursorParam)

	switch {
	case errors.Is(err, messagesrepo.ErrInvalidCursor):
		return Response{}, ErrInvalidCursor
	case err != nil:
		return Response{}, fmt.Errorf("failed to get messages: %v", err)
	}

	resp := Response{}

	if respCursor != nil {
		resp.NextCursor, err = cursor.Encode(respCursor)
		if err != nil {
			return Response{}, fmt.Errorf("failed to encode cursor: %v", err)
		}
	}

	resp.Messages = make([]Message, 0, len(messages))

	for _, msg := range messages {
		resp.Messages = append(resp.Messages, Message{
			ID:         msg.ID,
			AuthorID:   msg.AuthorID,
			Body:       msg.Body,
			CreatedAt:  msg.CreatedAt,
			IsReceived: msg.IsVisibleForManager && !msg.IsBlocked,
			IsBlocked:  msg.IsBlocked,
			IsService:  msg.IsService,
		})
	}

	return resp, nil
}
