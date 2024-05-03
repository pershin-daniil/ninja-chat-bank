package sendclientmessagejob

import (
	"errors"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func MarshalPayload(messageID types.MessageID) (string, error) {
	if messageID.IsZero() {
		return "", errors.New("empty message id")
	}

	return messageID.String(), nil
}
