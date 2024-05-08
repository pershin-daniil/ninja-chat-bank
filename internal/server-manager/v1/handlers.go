package managerv1

import (
	"context"
	"fmt"

	canreceiveproblems "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/can-receive-problems"
)

var _ ServerInterface = (*Handlers)(nil)

//go:generate mockgen -source=$GOFILE -destination=mocks/handlers_mocks.gen.go -package=managerv1mocks

type canReceiveProblemsUseCase interface {
	Handle(ctx context.Context, req canreceiveproblems.Request) (canreceiveproblems.Response, error)
}

//go:generate options-gen -out-filename=handlers_options.gen.go -from-struct=Options
type Options struct {
	canReceiveProblemsUseCase canReceiveProblemsUseCase `option:"mandatory" validate:"required"`
}

type Handlers struct {
	Options
}

func NewHandlers(opts Options) (Handlers, error) {
	if err := opts.Validate(); err != nil {
		return Handlers{}, fmt.Errorf("failed to validate options managerv1: %v", err)
	}

	return Handlers{Options: opts}, nil
}
