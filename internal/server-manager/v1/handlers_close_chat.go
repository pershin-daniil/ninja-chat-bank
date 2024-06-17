package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/pershin-daniil/ninja-chat-bank/internal/errors"
	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	closechat "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/close-chat"
)

func (h Handlers) PostCloseChat(eCtx echo.Context, params PostCloseChatParams) error {
	var httpRequest CloseChatRequest
	if err := eCtx.Bind(&httpRequest); err != nil {
		return fmt.Errorf("%w: %v", echo.ErrBadRequest, err)
	}

	req := closechat.Request{ChatID: httpRequest.ChatId, ManagerID: middlewares.MustUserID(eCtx), RequestID: params.XRequestID}

	err := h.closeChatUseCase.Handle(eCtx.Request().Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, closechat.ErrNoActiveProblemInChat):
			return internalerrors.NewServerError(int(ErrorCodeNoActiveProblemInChat), err.Error(), err)
		case errors.Is(err, closechat.ErrInvalidRequest):
			return fmt.Errorf("%w: %v", echo.ErrBadRequest, err)
		}
		return fmt.Errorf("%w: %v", echo.ErrInternalServerError, err)
	}

	return eCtx.JSON(http.StatusOK, CloseChatResponse{
		Data:  new(any),
		Error: nil,
	})
}
