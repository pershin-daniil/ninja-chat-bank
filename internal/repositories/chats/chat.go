package chatsrepo

import (
	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type Chat struct {
	ID       types.ChatID
	ClientID types.UserID
}

func adaptStoreChat(c *store.Chat) Chat {
	return Chat{
		ID:       c.ID,
		ClientID: c.ClientID,
	}
}

func adaptStoreChats(c []*store.Chat) []Chat {
	chats := make([]Chat, len(c))
	for i, chat := range c {
		chats[i] = adaptStoreChat(chat)
	}
	return chats
}
