package managerload

import (
	"context"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/manager_load_mocks.gen.go -package=managerloadmocks
type problemsRepository interface {
	GetManagerOpenProblemsCount(ctx context.Context, managerID types.UserID) (int, error)
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	maxProblemsAtTime int                `option:"mandatory" validate:"gte=1,lte=30"`
	problemsRepo      problemsRepository `option:"mandatory" validate:"required"`
}

type Service struct {
	Options
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate options managerload: %v", err)
	}

	return &Service{opts}, nil
}
