package errhandler

import (
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
)

type Response struct {
	Error clientv1.Error `json:"error"`
}

var ResponseBuilder = func(code int, msg string, details string) any {
	e := clientv1.Error{
		Code:    code,
		Message: msg,
	}

	if details != "" {
		e.Details = &details
	}

	return Response{
		Error: e,
	}
}
