package canreceiveproblems

import (
	"context"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/usecase_mock.gen.go -package=canreceiveproblemsmocks

type managerLoadService interface {
	CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error)
}

type managerPool interface {
	Contains(ctx context.Context, managerID types.UserID) (bool, error)
}

//go:generate options-gen -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	mLoadSvc managerLoadService `option:"mandatory" validate:"required"`
	mPool    managerPool        `option:"mandatory" validate:"required"`
}

type UseCase struct {
	Options
}

func New(opts Options) (UseCase, error) {
	if err := opts.Validate(); err != nil {
		return UseCase{}, fmt.Errorf("failed to validate options canreceiveproblems: %v", err)
	}

	return UseCase{opts}, nil
}

func (u UseCase) Handle(ctx context.Context, req Request) (Response, error) {
	if err := req.Validate(); err != nil {
		return Response{}, fmt.Errorf("failed to validate request: %v", err)
	}

	ok, err := u.mPool.Contains(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("failed to get info from manager pool: %v", err)
	}

	// Manager is already present in the pool.
	if ok {
		return Response{Result: false}, nil
	}

	ok, err = u.mLoadSvc.CanManagerTakeProblem(ctx, req.ManagerID)
	if err != nil {
		return Response{}, fmt.Errorf("failed to get info from manager load: %v", err)
	}

	return Response{Result: ok}, nil
}
