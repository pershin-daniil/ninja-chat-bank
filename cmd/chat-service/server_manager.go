package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server-client/errhandler"
	managerload "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-load"
	managerpool "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-pool"
)

const nameServerManager = "server-manager"

func initServerManager(
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
	lg := zap.L().Named(nameServerClient)

	errHandler, err := errhandler.New(errhandler.NewOptions(lg, isProduction, errhandler.ResponseBuilder))
	if err != nil {
		return nil, fmt.Errorf("failed to create errorHandler: %v", err)
	}

	srv, err := server.New(server.NewOptions(
		lg,
		addr,
		allowOrigins,
		v1Swagger,

		func(g *echo.Group) {},
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
