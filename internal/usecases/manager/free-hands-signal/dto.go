package freehandssignal

import (
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

type Request struct {
	ID        types.RequestID `validate:"required"`
	ManagerID types.UserID    `validate:"required"`
}

func (r Request) Validate() (err error) {
	return validator.Validator.Struct(r)
}

type Response struct{}
