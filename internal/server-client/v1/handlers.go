package clientv1

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	gethistory "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/get-history"
	sendmessage "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/send-message"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/handlers_mocks.gen.go -package=clientv1mocks

type getHistoryUseCase interface {
	Handle(ctx context.Context, req gethistory.Request) (gethistory.Response, error)
}

type sendMessageUseCase interface {
	Handle(ctx context.Context, req sendmessage.Request) (sendmessage.Response, error)
}

//go:generate options-gen -out-filename=handlers_options.gen.go -from-struct=Options
type Options struct {
	logger             *zap.Logger        `option:"mandatory" validate:"required"`
	getHistoryUseCase  getHistoryUseCase  `option:"mandatory" validate:"required"`
	sendMessageUseCase sendMessageUseCase `option:"mandatory" validate:"required"`
}

type Handlers struct {
	Options
}

func NewHandlers(opts Options) (Handlers, error) {
	if err := opts.Validate(); err != nil {
		return Handlers{}, fmt.Errorf("failed to validate options clientv1: %v", err)
	}

	return Handlers{Options: opts}, nil
}
