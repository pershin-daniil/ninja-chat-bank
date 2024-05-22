package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	errs "github.com/pershin-daniil/ninja-chat-bank/internal/errors"
	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	canreceiveproblems "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/can-receive-problems"
	freehands "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/free-hands"
)

const (
	ErrorCodeManagerOverloaded = 5000
	ManagerOverloadedError     = "manager overloaded"
)

func (h Handlers) PostGetFreeHandsBtnAvailability(eCtx echo.Context, params PostGetFreeHandsBtnAvailabilityParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)
	req := canreceiveproblems.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	}

	resp, err := h.canReceiveProblemsUseCase.Handle(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to handle can receive problems usecase: %v", err)
	}

	err = eCtx.JSON(http.StatusOK, GetFreeHandsBtnAvailabilityResponse{
		Data: map[string]any{"available": resp.Result},
	})
	if err != nil {
		return fmt.Errorf("failed to send response GetFreeHandsBtnAvailabilityResponse: %v", err)
	}

	return nil
}

func (h Handlers) PostFreeHands(eCtx echo.Context, params PostFreeHandsParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)
	req := freehands.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	}

	err := h.freeHandsUseCase.Handle(ctx, req)
	switch {
	case errors.Is(err, freehands.ErrManagerOverloaded):
		return errs.NewServerError(ErrorCodeManagerOverloaded, ManagerOverloadedError, err)
	case err != nil:
		return fmt.Errorf("failed to handle freeHandsUseCase: %v", err)
	}

	if err = eCtx.JSON(http.StatusOK, FreeHandsResponse{Data: nil}); err != nil {
		return fmt.Errorf("failed to send response FreeHandsResponse: %v", err)
	}

	return nil
}
