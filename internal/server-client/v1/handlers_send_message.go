package clientv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/pershin-daniil/ninja-chat-bank/internal/errors"
	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	sendmessage "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/send-message"
)

func (h Handlers) PostSendMessage(eCtx echo.Context, params PostSendMessageParams) error {
	ctx := eCtx.Request().Context()
	clientID := middlewares.MustUserID(eCtx)

	var req SendMessageRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("%w: %v", echo.ErrBadRequest, err)
	}

	response, err := h.sendMessage.Handle(ctx, sendmessage.Request{
		ID:          params.XRequestID,
		ClientID:    clientID,
		MessageBody: req.MessageBody,
	})
	if err != nil {
		if errors.Is(err, sendmessage.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		if errors.Is(err, sendmessage.ErrChatNotCreated) {
			return internalerrors.NewServerError(int(ErrorCodeCreateChatError), "create chat error", err)
		}

		if errors.Is(err, sendmessage.ErrProblemNotCreated) {
			return internalerrors.NewServerError(int(ErrorCodeCreateProblemError), "create problem error", err)
		}

		return fmt.Errorf("handle `send message` use case: %v", err)
	}

	return eCtx.JSON(http.StatusOK, SendMessageResponse{Data: &MessageHeader{
		AuthorId:  response.AuthorID.AsPointer(),
		CreatedAt: response.CreatedAt,
		Id:        response.MessageID,
	}})
}
