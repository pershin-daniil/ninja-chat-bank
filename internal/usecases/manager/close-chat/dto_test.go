package closechat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	closechat "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/close-chat"
)

func TestRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		request closechat.Request
		wantErr bool
	}{
		// Positive
		{
			name: "valid request",
			request: closechat.Request{
				ChatID:    types.NewChatID(),
				ManagerID: types.NewUserID(),
				RequestID: types.NewRequestID(),
			},
			wantErr: false,
		},

		// Negative
		{
			name: "empty chat id",
			request: closechat.Request{
				ChatID:    types.ChatIDNil,
				ManagerID: types.NewUserID(),
				RequestID: types.NewRequestID(),
			},
			wantErr: true,
		},
		{
			name: "empty manager id",
			request: closechat.Request{
				ChatID:    types.NewChatID(),
				ManagerID: types.UserIDNil,
				RequestID: types.NewRequestID(),
			},
			wantErr: true,
		},
		{
			name: "empty request id",
			request: closechat.Request{
				ChatID:    types.NewChatID(),
				ManagerID: types.NewUserID(),
				RequestID: types.RequestIDNil,
			},
			wantErr: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
