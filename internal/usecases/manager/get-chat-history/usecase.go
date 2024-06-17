package getchathistory

import (
	"context"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/cursor"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -typed -package=getchathistorymocks

type messagesRepository interface {
	GetProblemMessages(
		ctx context.Context,
		problemID types.ProblemID,
		pageSize int,
		cursor *messagesrepo.Cursor,
	) ([]messagesrepo.Message, *messagesrepo.Cursor, error)
}

type problemsRepo interface {
	GetUnresolvedProblem(ctx context.Context, chatID types.ChatID, managerID types.UserID) (*problemsrepo.Problem, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	messagesRepo messagesRepository `option:"mandatory" validate:"required"`
	problemsRepo problemsRepo       `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("options validate error: %v", err)
	}
	return UseCase{Options: opts}, nil
}

func (u UseCase) Handle(ctx context.Context, request Request) (Response, error) {
	if err := request.Validate(); err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	var cur *messagesrepo.Cursor
	if request.Cursor != "" {
		if err := cursor.Decode(request.Cursor, &cur); err != nil {
			return Response{}, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
		}
	}

	problem, err := u.problemsRepo.GetUnresolvedProblem(ctx, request.ChatID, request.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("get unresolved problem: %v", err)
	}

	messages, nextCur, err := u.messagesRepo.GetProblemMessages(ctx, problem.ID, request.PageSize, cur)
	if err != nil {
		return Response{}, fmt.Errorf("get problem messages: %v", err)
	}

	resp := Response{}
	if nextCur != nil {
		if resp.NextCursor, err = cursor.Encode(nextCur); err != nil {
			return Response{}, fmt.Errorf("cursor encode: %v", err)
		}
	}

	resp.Messages = make([]Message, len(messages))
	for i, m := range messages {
		resp.Messages[i] = Message{
			ID:        m.ID,
			AuthorID:  m.AuthorID,
			Body:      m.Body,
			CreatedAt: m.CreatedAt,
		}
	}
	return resp, nil
}
