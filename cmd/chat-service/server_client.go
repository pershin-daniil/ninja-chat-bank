package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	chatsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/chats"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server"
	"github.com/pershin-daniil/ninja-chat-bank/internal/server-client/errhandler"
	clientevents "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/events"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	inmemeventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream/in-mem"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	gethistory "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/get-history"
	sendmessage "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/client/send-message"
	websocketstream "github.com/pershin-daniil/ninja-chat-bank/internal/websocket-stream"
)

const nameServerClient = "server-client"

func initServerClient( //nolint:revive // https://giphy.com/gifs/5Zesu5VPNGJlm/fullscreen
	isProduction bool,
	addr string,
	allowOrigins []string,
	secWsProtocol string,
	eventStream *inmemeventstream.Service,
	v1Swagger *openapi3.T,

	client *keycloakclient.Client,
	resource string,
	role string,

	msgRepo *messagesrepo.Repo,
	chatRepo *chatsrepo.Repo,
	problemRepo *problemsrepo.Repo,

	outboxService *outbox.Service,

	db *store.Database,
) (*server.Server, error) {
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

	wsClientShutdown := make(chan struct{})
	wsClientUpgrader := websocketstream.NewUpgrader(
		allowOrigins,
		secWsProtocol,
	)

	wsClientHandler, err := websocketstream.NewHTTPHandler(
		websocketstream.NewOptions(
			zap.L(),
			eventStream,
			clientevents.Adapter{},
			websocketstream.JSONEventWriter{},
			wsClientUpgrader,
			wsClientShutdown,
		))
	if err != nil {
		return nil, fmt.Errorf("failed to init websocket client handler: %v", err)
	}

	srv, err := server.New(server.NewOptions(
		lg,
		addr,
		allowOrigins,
		v1Swagger,
		func(g *echo.Group) {
			clientv1.RegisterHandlers(g, v1Handlers)
		},
		client,
		resource,
		role,
		secWsProtocol,
		wsClientHandler,
		errHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to build server: %v", err)
	}

	return srv, nil
}
