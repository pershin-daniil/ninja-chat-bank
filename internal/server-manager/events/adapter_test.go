package managerevents_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managerevents "github.com/pershin-daniil/ninja-chat-bank/internal/server-manager/events"
	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func TestAdapter_Adapt(t *testing.T) {
	cases := []struct {
		name    string
		ev      eventstream.Event
		expJSON string
	}{
		{
			name: "smoke",
			ev: eventstream.NewNewChatEvent(
				types.MustParse[types.EventID]("d0ffbd36-bc30-11ed-8286-461e464ebed8"),
				types.MustParse[types.ChatID]("cb36a888-bc30-11ed-b843-461e464ebed8"),
				types.MustParse[types.UserID]("fec01fe8-483b-4cad-a0f6-ad0d431b433f"),
				types.MustParse[types.RequestID]("cee5f290-bc30-11ed-b7fe-461e464ebed8"),
				false,
			),
			expJSON: `{
				"eventId": "d0ffbd36-bc30-11ed-8286-461e464ebed8",
				"eventType": "NewChatEvent",
				"chatId": "cb36a888-bc30-11ed-b843-461e464ebed8",
				"clientId": "fec01fe8-483b-4cad-a0f6-ad0d431b433f",
				"requestId": "cee5f290-bc30-11ed-b7fe-461e464ebed8",
				"canTakeMoreProblems": false
			}`,
		},
		{
			name: "ChatClosedEvent",
			ev: eventstream.NewChatClosedEvent(
				types.MustParse[types.EventID]("d0ffbd36-bc30-11ed-8286-461e464ebed8"),
				types.MustParse[types.ChatID]("cb36a888-bc30-11ed-b843-461e464ebed8"),
				types.MustParse[types.RequestID]("cee5f290-bc30-11ed-b7fe-461e464ebed8"),
				true,
			),
			expJSON: `{
				"eventId": "d0ffbd36-bc30-11ed-8286-461e464ebed8",
				"eventType": "ChatClosedEvent",
				"chatId": "cb36a888-bc30-11ed-b843-461e464ebed8",
				"requestId": "cee5f290-bc30-11ed-b7fe-461e464ebed8",	
				"canTakeMoreProblems": true	
			}`,
		},
		{
			name: "NewMessageEvent",
			ev: eventstream.NewNewMessageEvent(
				types.MustParse[types.EventID]("d0ffbd36-bc30-11ed-8286-461e464ebed8"),
				types.MustParse[types.RequestID]("cee5f290-bc30-11ed-b7fe-461e464ebed8"),
				types.MustParse[types.ChatID]("cb36a888-bc30-11ed-b843-461e464ebed8"),
				types.MustParse[types.MessageID]("cb36a888-bc30-11ed-b843-1234567ebed8"),
				types.MustParse[types.UserID]("fec01fe8-483b-4cad-a0f6-ad0d431b433f"),
				time.Unix(1, 1).UTC(),
				"Where is my money?",
				false,
			),
			expJSON: `{
				"eventId": "d0ffbd36-bc30-11ed-8286-461e464ebed8",
				"eventType": "NewMessageEvent",
				"requestId": "cee5f290-bc30-11ed-b7fe-461e464ebed8",
				"chatId": "cb36a888-bc30-11ed-b843-461e464ebed8",
				"messageId": "cb36a888-bc30-11ed-b843-1234567ebed8",
				"authorId": "fec01fe8-483b-4cad-a0f6-ad0d431b433f",
				"body": "Where is my money?",
				"createdAt": "1970-01-01T00:00:01.000000001Z"
			}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			adapted, err := managerevents.Adapter{}.Adapt(tt.ev)
			require.NoError(t, err)

			raw, err := json.Marshal(adapted)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expJSON, string(raw))
		})
	}
}
