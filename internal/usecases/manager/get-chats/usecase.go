package getchats

import (
	"context"
	"fmt"

	chatsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/chats"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -typed -package=getchatsmocks

type chatsRepository interface {
	GetOpenProblemChatsForManager(ctx context.Context, managerID types.UserID) ([]chatsrepo.Chat, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	chatsRepo chatsRepository `option:"mandatory" validate:"required"`
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

	chats, err := u.chatsRepo.GetOpenProblemChatsForManager(ctx, request.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("get chats: %v", err)
	}

	response := Response{Chats: make([]Chat, len(chats))}
	for i, chat := range chats {
		response.Chats[i] = Chat{ID: chat.ID, ClientID: chat.ClientID}
	}

	return response, nil
}
