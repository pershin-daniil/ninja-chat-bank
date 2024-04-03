package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	serverclient "github.com/pershin-daniil/ninja-chat-bank/internal/server-client"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
)

const nameServerClient = "server-client"

func initServerClient(
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,
) (*serverclient.Server, error) {
	lg := zap.L().Named(nameServerClient)

	v1Handlers, err := clientv1.NewHandlers(clientv1.NewOptions(lg))
	if err != nil {
		return nil, fmt.Errorf("failed to create v1 handlers: %v", err)
	}

	srv, err := serverclient.New(serverclient.NewOptions(
		lg,
		addr,
		allowOrigins,
		v1Swagger,
		v1Handlers,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to build server: %v", err)
	}

	return srv, nil
}
