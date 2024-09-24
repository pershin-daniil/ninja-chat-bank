package closechat

import "errors"

var (
	ErrInvalidRequest        = errors.New("invalid request")
	ErrNoActiveProblemInChat = errors.New("no active problem in chat")
)
