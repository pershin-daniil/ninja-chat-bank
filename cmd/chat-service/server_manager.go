package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server-client/errhandler"
	managerv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-manager/v1"
	managerload "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-load"
	managerpool "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-pool"
	canreceiveproblems "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/can-receive-problems"
	freehands "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/free-hands"
)

const nameServerManager = "server-manager"

func initServerManager( //nolint:revive // https://giphy.com/gifs/5Zesu5VPNGJlm/fullscreen
	isProduction bool,
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	client *keycloakclient.Client,
	resource string,
	role string,

	managerLoad *managerload.Service,
	managerPool managerpool.Pool,
) (*server.Server, error) {
	lg := zap.L().Named(nameServerManager)

	canReceiveProblemsUseCase, err := canreceiveproblems.New(canreceiveproblems.NewOptions(
		managerPool,
		managerLoad,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to init canReceiveProblemsUseCase: %v", err)
	}

	freeHandsUseCase, err := freehands.New(freehands.NewOptions(
		managerPool,
		managerLoad,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to init freeHandsUseCase: %v", err)
	}

	v1Handlers, err := managerv1.NewHandlers(managerv1.NewOptions(lg, canReceiveProblemsUseCase, freeHandsUseCase))
	if err != nil {
		return nil, fmt.Errorf("failed to init manager handlers: %v", err)
	}

	errHandler, err := errhandler.New(errhandler.NewOptions(lg, isProduction, errhandler.ResponseBuilder))
	if err != nil {
		return nil, fmt.Errorf("failed to create errorHandler: %v", err)
	}

	srv, err := server.New(server.NewOptions(
		lg,
		addr,
		allowOrigins,
		v1Swagger,
		func(g *echo.Group) {
			managerv1.RegisterHandlers(g, v1Handlers)
		},
		client,
		resource,
		role,
		errHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to build manager server: %v", err)
	}

	return srv, nil
}
