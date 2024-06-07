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
	jobsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/jobs"
	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	clientevents "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/events"
	clientv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-client/v1"
	serverdebug "github.com/pershin-daniil/ninja-chat-bank/internal/server-debug"
	managerv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-manager/v1"
	afcverdictsprocessor "github.com/pershin-daniil/ninja-chat-bank/internal/services/afc-verdicts-processor"
	inmemeventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream/in-mem"
	managerload "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-load"
	inmemmanagerpool "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-pool/in-mem"
	msgproducer "github.com/pershin-daniil/ninja-chat-bank/internal/services/msg-producer"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox"
	clientmessageblockedjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/client-message-blocked"
	clientmessagesentjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/client-message-sent"
	sendclientmessagejob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/send-client-message"
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
		logger.WithSentryDSN(cfg.Sentry.DSN),
		logger.WithSentryEnv(cfg.Global.Env),
	)); err != nil {
		return fmt.Errorf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	clientSwagger, err := clientv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to get client swagger: %v", err)
	}

	managerSwagger, err := managerv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to get client swagger: %v", err)
	}

	eventsSwagger, err := clientevents.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to get events swagger: %v", err)
	}

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(
		cfg.Servers.Debug.Addr,
		clientSwagger,
		managerSwagger,
		eventsSwagger,
	))
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
			zap.L().Warn("failed to close psqlClient", zap.Error(e))
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

	jobsRepo, err := jobsrepo.New(jobsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("failed to init jobs repo: %v", err)
	}

	outBox, err := outbox.New(outbox.NewOptions(
		cfg.Services.OutboxConfig.Workers,
		cfg.Services.OutboxConfig.IdleTime,
		cfg.Services.OutboxConfig.ReserveFor,
		jobsRepo,
		db,
	))
	if err != nil {
		return fmt.Errorf("failed to init outbox service: %v", err)
	}

	msgProducer, err := msgproducer.New(msgproducer.NewOptions(
		msgproducer.NewKafkaWriter(
			cfg.Services.MsgProducerConfig.Brokers,
			cfg.Services.MsgProducerConfig.Topic,
			cfg.Services.MsgProducerConfig.BatchSize,
		)))
	if err != nil {
		return fmt.Errorf("failed to init message producer: %v", err)
	}

	eventStream := inmemeventstream.New()
	defer func() {
		if e := eventStream.Close(); err != nil {
			zap.L().Warn("failed to close inmem event stream", zap.Error(e))
		}
	}()

	for _, j := range []outbox.Job{
		sendclientmessagejob.Must(sendclientmessagejob.NewOptions(msgProducer, msgRepo, eventStream)),
		clientmessageblockedjob.Must(clientmessageblockedjob.NewOptions(msgRepo, eventStream)),
		clientmessagesentjob.Must(clientmessagesentjob.NewOptions(msgRepo, eventStream)),
	} {
		outBox.MustRegisterJob(j)
	}

	// AFC verdict processor
	afcVerdictProcessor, err := afcverdictsprocessor.New(afcverdictsprocessor.NewOptions(
		cfg.Services.AFCVerdictProcessorConfig.Brokers,
		cfg.Services.AFCVerdictProcessorConfig.Consumers,
		cfg.Services.AFCVerdictProcessorConfig.ConsumerGroup,
		cfg.Services.AFCVerdictProcessorConfig.VerdictsTopic,
		afcverdictsprocessor.NewKafkaReader,
		afcverdictsprocessor.NewKafkaDLQWriter(
			cfg.Services.AFCVerdictProcessorConfig.Brokers,
			cfg.Services.AFCVerdictProcessorConfig.VerdictsDlqTopic,
		),
		db,
		msgRepo,
		outBox,
		afcverdictsprocessor.WithVerdictsSignKey(cfg.Services.AFCVerdictProcessorConfig.VerdictsSigningPublicKey),
		afcverdictsprocessor.WithProcessBatchSize(cfg.Services.AFCVerdictProcessorConfig.BatchSize),
	))
	if err != nil {
		return fmt.Errorf("AFC verdict processor: %v", err)
	}

	srvClient, err := initServerClient(
		cfg.IsProduction(),
		cfg.Servers.Client.Addr,
		cfg.Servers.Client.AllowOrigins,
		cfg.Servers.Client.SecWSProtocol,
		eventStream,
		clientSwagger,
		kcClient,
		cfg.Servers.Client.RequiredAccess.Resource,
		cfg.Servers.Client.RequiredAccess.Role,
		msgRepo,
		chatRepo,
		problemRepo,
		outBox,
		db,
	)
	if err != nil {
		return fmt.Errorf("failed to init server: %v", err)
	}

	mngPool := inmemmanagerpool.New()
	mngLoad, err := managerload.New(managerload.NewOptions(
		cfg.Services.ManagerLoadConfig.MaxProblems,
		problemRepo,
	))
	if err != nil {
		return fmt.Errorf("failed to init load service: %v", err)
	}

	srvManager, err := initServerManager(
		cfg.IsProduction(),
		cfg.Servers.Manager.Addr,
		cfg.Servers.Manager.AllowOrigins,
		cfg.Servers.Manager.SecWSProtocol,
		eventStream,
		managerSwagger,
		kcClient,
		cfg.Servers.Manager.RequiredAccess.Resource,
		cfg.Servers.Manager.RequiredAccess.Role,
		mngLoad,
		mngPool,
	)
	if err != nil {
		return fmt.Errorf("failed to init manager server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvDebug.Run(ctx) })
	eg.Go(func() error { return srvClient.Run(ctx) })
	eg.Go(func() error { return srvManager.Run(ctx) })

	// Run services.
	eg.Go(func() error { return outBox.Run(ctx) })
	eg.Go(func() error { return afcVerdictProcessor.Run(ctx) })

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("failed to wait app stop: %v", err)
	}

	return nil
}
