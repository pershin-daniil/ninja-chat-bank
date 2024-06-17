package closechatjob_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	closechatjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/close-chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func TestMarshal(t *testing.T) {
	// Valid input.
	t.Run("valid input", func(t *testing.T) {
		payload := closechatjob.Payload{
			RequestID: types.NewRequestID(),
			ManagerID: types.NewUserID(),
			MessageID: types.NewMessageID(),
			ClientID:  types.NewUserID(),
		}

		strPayload, err := closechatjob.Marshal(payload)

		require.NoError(t, err)
		assert.NotEmpty(t, strPayload)
	})

	// Invalid input.
	t.Run("empty request id", func(t *testing.T) {
		payload := closechatjob.Payload{
			RequestID: types.RequestIDNil,
			ManagerID: types.NewUserID(),
			MessageID: types.NewMessageID(),
			ClientID:  types.NewUserID(),
		}

		strPayload, err := closechatjob.Marshal(payload)

		require.Error(t, err)
		assert.Empty(t, strPayload)
	})

	t.Run("empty manager id", func(t *testing.T) {
		payload := closechatjob.Payload{
			RequestID: types.NewRequestID(),
			ManagerID: types.UserIDNil,
			MessageID: types.NewMessageID(),
			ClientID:  types.NewUserID(),
		}

		strPayload, err := closechatjob.Marshal(payload)

		require.Error(t, err)
		assert.Empty(t, strPayload)
	})

	t.Run("empty message id", func(t *testing.T) {
		payload := closechatjob.Payload{
			RequestID: types.NewRequestID(),
			ManagerID: types.NewUserID(),
			MessageID: types.MessageIDNil,
			ClientID:  types.NewUserID(),
		}

		strPayload, err := closechatjob.Marshal(payload)

		require.Error(t, err)
		assert.Empty(t, strPayload)
	})

	t.Run("empty client id", func(t *testing.T) {
		payload := closechatjob.Payload{
			RequestID: types.NewRequestID(),
			ManagerID: types.NewUserID(),
			MessageID: types.NewMessageID(),
			ClientID:  types.UserIDNil,
		}

		strPayload, err := closechatjob.Marshal(payload)

		require.Error(t, err)
		assert.Empty(t, strPayload)
	})
}

func TestUnmarshal(t *testing.T) {
	// Valid input.
	t.Run("valid input", func(t *testing.T) {
		requestID := types.NewRequestID()
		managerID := types.NewUserID()
		messageID := types.NewMessageID()
		clientID := types.NewUserID()
		payload := fmt.Sprintf(
			`{"requestId":"%s","managerId":"%s","messageId":"%s","clientId":"%s"}`,
			requestID, managerID, messageID, clientID,
		)

		p, err := closechatjob.Unmarshal(payload)

		require.NoError(t, err)
		assert.NotEmpty(t, p)
		assert.Equal(t, requestID, p.RequestID)
		assert.Equal(t, managerID, p.ManagerID)
	})

	// Invalid input.
	t.Run("empty request id", func(t *testing.T) {
		requestID := types.RequestIDNil
		managerID := types.NewUserID()
		messageID := types.NewMessageID()
		clientID := types.NewUserID()
		payload := fmt.Sprintf(
			`{"requestId":"%s","managerId":"%s","messageId":"%s","clientId":"%s"}`,
			requestID, managerID, messageID, clientID,
		)

		p, err := closechatjob.Unmarshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("empty manager id", func(t *testing.T) {
		requestID := types.NewRequestID()
		managerID := types.UserIDNil
		messageID := types.NewMessageID()
		clientID := types.NewUserID()
		payload := fmt.Sprintf(
			`{"requestId":"%s","managerId":"%s","messageId":"%s","clientId":"%s"}`,
			requestID, managerID, messageID, clientID,
		)

		p, err := closechatjob.Unmarshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("empty message id", func(t *testing.T) {
		requestID := types.NewRequestID()
		managerID := types.NewUserID()
		messageID := types.MessageIDNil
		clientID := types.NewUserID()
		payload := fmt.Sprintf(
			`{"requestId":"%s","managerId":"%s","messageId":"%s","clientId":"%s"}`,
			requestID, managerID, messageID, clientID,
		)

		p, err := closechatjob.Unmarshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("empty payload", func(t *testing.T) {
		p, err := closechatjob.Unmarshal("")

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		p, err := closechatjob.Unmarshal("{")

		require.Error(t, err)
		assert.Empty(t, p)
	})
}
