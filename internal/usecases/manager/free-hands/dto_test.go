package freehands_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	freehands "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/free-hands"
)

func TestRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		request freehands.Request
		wantErr bool
	}{
		// Positive.
		{
			name: "valid request",
			request: freehands.Request{
				ID:        types.NewRequestID(),
				ManagerID: types.NewUserID(),
			},
			wantErr: false,
		},

		// Negative.
		{
			name: "not valid request 1",
			request: freehands.Request{
				ManagerID: types.NewUserID(),
			},
			wantErr: true,
		},
		{
			name: "not valid request 2",
			request: freehands.Request{
				ID: types.NewRequestID(),
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
