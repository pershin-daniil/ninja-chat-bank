package managerassignedtoproblemjob

import (
	"encoding/json"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/internal/validator"
)

type Payload struct {
	MessageID types.MessageID `json:"messageId" validate:"required"`
	ManagerID types.UserID    `json:"managerId" validate:"required"`
	ClientID  types.UserID    `json:"clientId" validate:"required"`
}

func Marshal(payload Payload) (string, error) {
	if err := validator.Validator.Struct(payload); err != nil {
		return "", fmt.Errorf("validate: %v", err)
	}

	result, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("json marshal: %v", err)
	}
	return string(result), nil
}

func Unmarshal(payload string) (Payload, error) {
	var p Payload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return Payload{}, fmt.Errorf("json unmarshal: %v", err)
	}

	if err := validator.Validator.Struct(p); err != nil {
		return Payload{}, fmt.Errorf("validate: %v", err)
	}

	return p, nil
}
