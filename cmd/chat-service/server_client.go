package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	serverclient "github.com/pershin-daniil/ninja-chat-bank/internal/server-client"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server-client/errhandler"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	gethistory "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/get-history"
)

const nameServerClient = "server-client"

func initServerClient(
	isProduction bool,
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	client *keycloakclient.Client,
	resource string,
	role string,
	msgRepo *messagesrepo.Repo,
) (*serverclient.Server, error) {
	lg := zap.L().Named(nameServerClient)

	getHistoryUseCase, err := gethistory.New(gethistory.NewOptions(msgRepo))
	if err != nil {
		return nil, fmt.Errorf("failed to create getHistoryUsrCase: %v", err)
	}

	v1Handlers, err := clientv1.NewHandlers(clientv1.NewOptions(lg, getHistoryUseCase))
	if err != nil {
		return nil, fmt.Errorf("failed to create v1 handlers: %v", err)
	}

	errHandler, err := errhandler.New(errhandler.NewOptions(lg, isProduction, errhandler.ResponseBuilder))
	if err != nil {
		return nil, fmt.Errorf("failed to create errorHandler: %v", err)
	}

	srv, err := serverclient.New(serverclient.NewOptions(
		lg,
		addr,
		allowOrigins,
		v1Swagger,
		v1Handlers,
		client,
		resource,
		role,
		errHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to build server: %v", err)
	}

	return srv, nil
}
