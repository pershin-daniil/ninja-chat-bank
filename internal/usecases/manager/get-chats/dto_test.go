package getchats_test

import (
	"testing"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	getchats "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/get-chats"
)

func TestRequest_Validate(t *testing.T) {
	type fields struct {
		ID        types.RequestID
		ManagerID types.UserID
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "empty id",
			fields:  fields{ID: types.RequestIDNil, ManagerID: types.NewUserID()},
			wantErr: true,
		},
		{
			name:    "empty manager id",
			fields:  fields{ID: types.NewRequestID(), ManagerID: types.UserIDNil},
			wantErr: true,
		},
		{
			name:    "no error",
			fields:  fields{ID: types.NewRequestID(), ManagerID: types.NewUserID()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := getchats.Request{
				ID:        tt.fields.ID,
				ManagerID: tt.fields.ManagerID,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
