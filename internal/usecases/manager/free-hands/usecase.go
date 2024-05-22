package freehands

import (
	"context"
	"errors"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=freehandsmocks

var ErrManagerOverloaded = errors.New("manager overloaded")

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

type managerPool interface {
	Put(ctx context.Context, managerID types.UserID) error
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	mngPool managerPool        `option:"mandatory" validate:"required"`
	mngLoad managerLoadService `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("failed to validate options freehands: %v", err)
	}

	return UseCase{opts}, nil
}

func (u UseCase) Handle(ctx context.Context, req Request) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("failed to validate request: %v", err)
	}

	can, err := u.mngLoad.CanManagerTakeProblem(ctx, req.ManagerID)
	if err != nil {
		return fmt.Errorf("failed to get info from manager load: %v", err)
	}

	if !can {
		return ErrManagerOverloaded
	}

	if err = u.mngPool.Put(ctx, req.ManagerID); err != nil {
		return fmt.Errorf("failed to put manager to pool: %v", err)
	}

	return nil
}
