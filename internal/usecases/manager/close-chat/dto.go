package closechat

import (
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

type Request struct {
	ChatID    types.ChatID    `validate:"required"`
	ManagerID types.UserID    `validate:"required"`
	RequestID types.RequestID `validate:"required"`
}

func (r Request) Validate() error {
	if err := validator.Validator.Struct(r); err != nil {
		return fmt.Errorf("validate: %v", err)
	}
	return nil
}
