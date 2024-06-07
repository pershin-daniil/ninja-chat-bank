package eventstream

import (
	"context"
	"io"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type EventStream interface {
	io.Closer
	Subscribe(ctx context.Context, userID types.UserID) (<-chan Event, error)
	Publish(ctx context.Context, userID types.UserID, event Event) error
}
