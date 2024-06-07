package inmemeventstream

import (
	"context"

	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type clients struct {
	items map[types.UserID][]*client
}

type client struct {
	ctx context.Context
	ch  chan eventstream.Event

	id     types.EventClientID
	userID types.UserID
}

func (c *clients) add(ctx context.Context, userID types.UserID) *client {
	client := &client{
		ctx: ctx,
		ch:  make(chan eventstream.Event, 1024),

		id:     types.NewEventClientID(),
		userID: userID,
	}

	c.items[userID] = append(c.items[userID], client)

	return client
}

func (c *clients) get(userID types.UserID) []*client {
	return c.items[userID]
}

func (c *clients) remove(client *client) {
	clients, exist := c.items[client.userID]
	if !exist {
		return
	}

	for i := 0; i < len(clients); i++ {
		if clients[i].id == client.id {
			c.items[client.userID] = remove(clients, i)
			close(client.ch)
		}
	}

	if len(c.items[client.userID]) == 0 {
		delete(c.items, client.userID)
	}
}

func remove(s []*client, i int) []*client {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func newClients() *clients {
	return &clients{
		items: make(map[types.UserID][]*client),
	}
}
