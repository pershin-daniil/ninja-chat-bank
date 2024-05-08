package freehands

import (
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type Request struct {
	ID        types.RequestID `validate:"required"`
	ManagerID types.UserID    `validate:"required"`
}

func (r Request) Validate() (err error) {
	if err = r.ID.Validate(); err != nil {
		return fmt.Errorf("failed to validate request id: %v", err)
	}

	if err = r.ManagerID.Validate(); err != nil {
		return fmt.Errorf("failed to validate request manager id: %v", err)
	}

	return nil
}

type Response struct{}
