package managerassignedtoproblemjob_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managerassignedtoproblemjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/manager-assigned-to-problem"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func TestMarshal_Smoke(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		payload := managerassignedtoproblemjob.Payload{
			MessageID: types.NewMessageID(),
			ManagerID: types.NewUserID(),
			ClientID:  types.NewUserID(),
		}

		p, err := managerassignedtoproblemjob.Marshal(payload)

		require.NoError(t, err)
		assert.NotEmpty(t, p)
	})

	t.Run("invalid input", func(t *testing.T) {
		payload := managerassignedtoproblemjob.Payload{
			MessageID: types.MessageID{},
			ManagerID: types.UserID{},
		}

		p, err := managerassignedtoproblemjob.Marshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})
}

func TestUnmarshal(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		messageID := types.NewMessageID()
		managerID := types.NewUserID()
		clientID := types.NewUserID()
		payload := fmt.Sprintf(`{"messageID":"%s","managerID":"%s","clientID":"%s"}`, messageID, managerID, clientID)

		p, err := managerassignedtoproblemjob.Unmarshal(payload)

		require.NoError(t, err)
		assert.NotEmpty(t, p)
		assert.Equal(t, messageID, p.MessageID)
		assert.Equal(t, managerID, p.ManagerID)
		assert.Equal(t, clientID, p.ClientID)
	})

	t.Run("empty message id", func(t *testing.T) {
		messageID := types.MessageIDNil
		managerID := types.NewUserID()
		clientID := types.NewUserID()
		payload := fmt.Sprintf(`{"messageID":"%s","managerID":"%s","clientID":"%s"}`, messageID, managerID, clientID)

		p, err := managerassignedtoproblemjob.Unmarshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("empty manager id", func(t *testing.T) {
		messageID := types.NewMessageID()
		managerID := types.UserIDNil
		clientID := types.NewUserID()

		payload := fmt.Sprintf(`{"messageID":"%s","managerID":"%s","clientID":"%s"}`, messageID, managerID, clientID)

		p, err := managerassignedtoproblemjob.Unmarshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("empty client id", func(t *testing.T) {
		messageID := types.NewMessageID()
		managerID := types.NewUserID()
		clientID := types.UserIDNil
		payload := fmt.Sprintf(`{"messageID":"%s","managerID":"%s","clientID":"%s"}`, messageID, managerID, clientID)

		p, err := managerassignedtoproblemjob.Unmarshal(payload)

		require.Error(t, err)
		assert.Empty(t, p)
	})

	t.Run("empty payload", func(t *testing.T) {
		p, err := managerassignedtoproblemjob.Unmarshal("")

		require.Error(t, err)
		assert.Empty(t, p)
	})
}
