package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/pershin-daniil/ninja-chat-bank/internal/config"
	"github.com/pershin-daniil/ninja-chat-bank/internal/logger"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	serverdebug "github.com/pershin-daniil/ninja-chat-bank/internal/server-debug"
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

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(cfg.Servers.Debug.Addr))
	if err != nil {
		return fmt.Errorf("failed to init debug server: %v", err)
	}

	swagger, err := clientv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to get swagger: %v", err)
	}

	srvClient, err := initServerClient(cfg.Servers.Client.Addr, cfg.Servers.Client.AllowOrigins, swagger)

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvDebug.Run(ctx) })
	eg.Go(func() error { return srvClient.Run(ctx) })

	// Run services.
	// Ждут своего часа.
	// ...

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("failed to wait app stop: %v", err)
	}

	return nil
}
