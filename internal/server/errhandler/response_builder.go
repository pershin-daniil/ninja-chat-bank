package errhandler

import (
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

type Response struct {
	Error clientv1.Error `json:"error"`
}

var ResponseBuilder = func(code int, msg string, details string) any {
	return Response{
		Error: clientv1.Error{
			Code:    clientv1.ErrorCode(code),
			Message: msg,
			Details: pointer.PtrWithZeroAsNil(details),
		},
	}
}
