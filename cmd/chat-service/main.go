package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	"github.com/pershin-daniil/ninja-chat-bank/internal/config"
	"github.com/pershin-daniil/ninja-chat-bank/internal/logger"
	chatsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/chats"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	serverdebug "github.com/pershin-daniil/ninja-chat-bank/internal/server-debug"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
)

var configPath = flag.String("config", "configs/config.toml", "Path to config file")

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}

func run() (errReturned error) {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.ParseAndValidate(*configPath)
	if err != nil {
		return fmt.Errorf("failed to parse and validate config %q: %v", *configPath, err)
	}

	if err = logger.Init(logger.NewOptions(
		cfg.Log.Level,
		logger.WithProductionMode(cfg.IsProduction()),
		logger.WithSentryDNS(cfg.Sentry.DSN),
	)); err != nil {
		return fmt.Errorf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	swagger, err := clientv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to get swagger: %v", err)
	}

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(cfg.Servers.Debug.Addr, swagger))
	if err != nil {
		return fmt.Errorf("failed to init debug server: %v", err)
	}

	kcClient, err := keycloakclient.New(keycloakclient.NewOptions(
		cfg.Clients.Keycloak.BasePath,
		cfg.Clients.Keycloak.Realm,
		cfg.Clients.Keycloak.ClientID,
		cfg.Clients.Keycloak.ClientSecret,
		cfg.IsProduction(),
		keycloakclient.WithDebugMode(cfg.Clients.Keycloak.DebugMode),
	))
	if err != nil {
		return fmt.Errorf("init keycloak client: %w", err)
	}

	psqlClient, err := store.NewPSQLClient(store.NewPSQLOptions(
		cfg.DB.Postgres.Addr,
		cfg.DB.Postgres.User,
		cfg.DB.Postgres.Password,
		cfg.DB.Postgres.Database,
		cfg.IsProduction(),
		store.WithDebugMode(cfg.DB.Postgres.DebugMode),
	))
	if err != nil {
		return fmt.Errorf("failed to init psql client: %v", err)
	}
	defer func() {
		if e := psqlClient.Close(); e != nil {
			zap.L().Warn("failed to close psqlClient: %v", zap.Error(e))
		}
	}()

	if err = psqlClient.Schema.Create(ctx); err != nil {
		return fmt.Errorf("failed to init schema: %v", err)
	}

	db := store.NewDatabase(psqlClient)

	msgRepo, err := messagesrepo.New(messagesrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("failed to init message repo: %v", err)
	}

	chatRepo, err := chatsrepo.New(chatsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("failed to init chat repo: %v", err)
	}

	problemRepo, err := problemsrepo.New(problemsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("failed to init problem repo: %v", err)
	}

	srvClient, err := initServerClient(
		cfg.IsProduction(),
		cfg.Servers.Client.Addr,
		cfg.Servers.Client.AllowOrigins,
		swagger,
		kcClient,
		cfg.Servers.Client.RequiredAccess.Resource,
		cfg.Servers.Client.RequiredAccess.Role,
		msgRepo,
		chatRepo,
		problemRepo,
		db,
	)
	if err != nil {
		return fmt.Errorf("failed to init server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvDebug.Run(ctx) })
	eg.Go(func() error { return srvClient.Run(ctx) })

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("failed to wait app stop: %v", err)
	}

	return nil
}
