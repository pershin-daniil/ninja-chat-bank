package sendmessage

import (
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

type Request struct {
	ID          types.RequestID `validate:"required"`
	ClientID    types.UserID    `validate:"required"`
	MessageBody string          `validate:"required,min=1,max=3000"`
}

func (r Request) Validate() error {
	return validator.Validator.Struct(r)
}

type Response struct {
	MessageID types.MessageID
	AuthorID  types.UserID
	CreatedAt time.Time
}
