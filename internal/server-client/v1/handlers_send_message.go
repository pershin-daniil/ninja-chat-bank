package clientv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	errs "github.com/pershin-daniil/ninja-chat-bank/internal/errors"
	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	sendmessage "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/send-message"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

func (h Handlers) PostSendMessage(eCtx echo.Context, params PostSendMessageParams) error {
	ctx := eCtx.Request().Context()
	clientID := middlewares.MustUserID(eCtx)

	var req SendMessageRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("%w: %v", echo.ErrBadRequest, err)
	}

	request := sendmessage.Request{
		ID:          params.XRequestID,
		ClientID:    clientID,
		MessageBody: req.MessageBody,
	}

	response, err := h.sendMessageUseCase.Handle(ctx, request)
	if err != nil {
		switch {
		case errors.Is(err, sendmessage.ErrInvalidRequest):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case errors.Is(err, sendmessage.ErrChatNotCreated):
			return errs.NewServerError(int(ErrorCodeCreateChatError), "failed to create chat", err)
		case errors.Is(err, sendmessage.ErrProblemNotCreated):
			return errs.NewServerError(int(ErrorCodeCreateProblemError), "failed to create problem", err)
		}

		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	err = eCtx.JSON(http.StatusOK, SendMessageResponse{
		Data: &MessageHeader{
			AuthorId:  pointer.PtrWithZeroAsNil(response.AuthorID),
			CreatedAt: response.CreatedAt,
			Id:        response.MessageID,
		},
		Error: nil,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", echo.ErrInternalServerError, err)
	}

	return nil
}
