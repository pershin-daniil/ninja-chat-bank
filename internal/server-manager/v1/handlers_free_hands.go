package managerv1

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	canreceiveproblems "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/can-receive-problems"
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
		Data: map[string]interface{}{"available": resp.Result},
	})
	if err != nil {
		return fmt.Errorf("failed to send response: %v", err)
	}

	return nil
}
