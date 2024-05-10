package problemsrepo

import (
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
)

//go:generate options-gen -out-filename=repo_options.gen.go -from-struct=Options
type Options struct {
	db *store.Database `option:"mandatory" validate:"required"`
}

type Repo struct {
	Options
}

func New(opts Options) (*Repo, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate options problemsrepo: %v", err)
	}

	return &Repo{Options: opts}, nil
}
