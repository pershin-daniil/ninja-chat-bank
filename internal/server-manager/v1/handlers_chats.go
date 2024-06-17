package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/pershin-daniil/ninja-chat-bank/internal/errors"
	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	getchathistory "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/get-chat-history"
	getchats "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/get-chats"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

func (h Handlers) PostGetChats(eCtx echo.Context, params PostGetChatsParams) error {
	req := getchats.Request{
		ID:        params.XRequestID,
		ManagerID: middlewares.MustUserID(eCtx),
	}

	result, err := h.getChatsUseCase.Handle(eCtx.Request().Context(), req)
	if err != nil {
		if errors.Is(err, getchats.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}
		return fmt.Errorf("%w: %v", echo.ErrInternalServerError, err)
	}

	chats := make([]Chat, len(result.Chats))
	for i, chat := range result.Chats {
		chats[i] = Chat{ChatId: chat.ID, ClientId: chat.ClientID}
	}

	return eCtx.JSON(http.StatusOK, GetChatsResponse{
		Data:  &ChatList{Chats: chats},
		Error: nil,
	})
}

func (h Handlers) PostGetChatHistory(eCtx echo.Context, params PostGetChatHistoryParams) error {
	var req GetChatHistoryRequest

	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("%w: %v", echo.ErrBadRequest, err)
	}

	request := getchathistory.Request{
		ID:        params.XRequestID,
		ManagerID: middlewares.MustUserID(eCtx),
		ChatID:    req.ChatId,
		PageSize:  pointer.Indirect(req.PageSize),
		Cursor:    pointer.Indirect(req.Cursor),
	}

	result, err := h.getChatHistoryUseCase.Handle(eCtx.Request().Context(), request)
	if err != nil {
		if errors.Is(err, getchathistory.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}
		return fmt.Errorf("%w: %v", echo.ErrInternalServerError, err)
	}

	messages := make([]Message, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = Message{
			AuthorId:  msg.AuthorID,
			Body:      msg.Body,
			CreatedAt: msg.CreatedAt,
			Id:        msg.ID,
		}
	}

	return eCtx.JSON(http.StatusOK, GetChatHistoryResponse{
		Data:  &MessagesPage{Messages: messages},
		Error: nil,
	})
}
