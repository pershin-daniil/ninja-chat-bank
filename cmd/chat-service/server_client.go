package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	chatsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/chats"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	serverclient "github.com/pershin-daniil/ninja-chat-bank/internal/server-client"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server-client/errhandler"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	gethistory "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/get-history"
	sendmessage "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/send-message"
)

const nameServerClient = "server-client"

func initServerClient( //nolint:revive // https://giphy.com/gifs/5Zesu5VPNGJlm/fullscreen
	isProduction bool,
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	client *keycloakclient.Client,
	resource string,
	role string,

	msgRepo *messagesrepo.Repo,
	chatRepo *chatsrepo.Repo,
	problemRepo *problemsrepo.Repo,

	outboxService *outbox.Service,

	db *store.Database,
) (*serverclient.Server, error) {
	lg := zap.L().Named(nameServerClient)

	getHistoryUseCase, err := gethistory.New(gethistory.NewOptions(msgRepo))
	if err != nil {
		return nil, fmt.Errorf("failed to create getHistoryUsrCase: %v", err)
	}

	sendMessageUseCase, err := sendmessage.New(sendmessage.NewOptions(chatRepo, msgRepo, outboxService, problemRepo, db))
	if err != nil {
		return nil, fmt.Errorf("failed to create sendMessageUseCase: %v", err)
	}

	v1Handlers, err := clientv1.NewHandlers(clientv1.NewOptions(lg, getHistoryUseCase, sendMessageUseCase))
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
