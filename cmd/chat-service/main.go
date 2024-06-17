package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"go.uber.org/multierr"
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
	managerevents "github.com/pershin-daniil/ninja-chat-bank/internal/server-manager/events"
	managerv1 "github.com/pershin-daniil/ninja-chat-bank/internal/server-manager/v1"
	afcverdictsprocessor "github.com/pershin-daniil/ninja-chat-bank/internal/services/afc-verdicts-processor"
	inmemeventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream/in-mem"
	managerload "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-load"
	inmemmanagerpool "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-pool/in-mem"
	managerscheduler "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-scheduler"
	msgproducer "github.com/pershin-daniil/ninja-chat-bank/internal/services/msg-producer"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox"
	clientmessageblockedjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/client-message-blocked"
	clientmessagesentjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/client-message-sent"
	managerassignedtoproblemjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/manager-assigned-to-problem"
	sendclientmessagejob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/send-client-message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
)

var configPath = flag.String("config", "configs/config.toml", "Path to config file")

func main() {
	if err := run(); err != nil {
		log.Fatalf("run app: %v", err)
	}
}

//nolint:gocyclo // ignore cyclomatic complexity
func run() (errReturned error) {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.ParseAndValidate(*configPath)
	if err != nil {
		return fmt.Errorf("parse and validate config %q: %v", *configPath, err)
	}

	if err = logger.Init(logger.NewOptions(
		cfg.Log.Level,
		logger.WithProductionMode(cfg.IsProduction()),
		logger.WithSentryDSN(cfg.Sentry.DSN),
		logger.WithSentryEnv(cfg.Global.Env),
	)); err != nil {
		return fmt.Errorf("init logger: %v", err)
	}
	defer logger.Sync()

	lg := zap.L().Named("main")

	// Storage.

	var storage *store.Client
	{
		switch s := cfg.Stores.Use; s {
		case "sqlite":
			storage, err = store.NewInMemorySQLiteClient()
		case "psql":
			if cfg.IsProduction() && cfg.Stores.PSQL.Debug {
				lg.Warn("psql client in the debug mode")
			}

			storage, err = store.NewPSQLClient(store.NewPSQLOptions(
				cfg.Stores.PSQL.Addr,
				cfg.Stores.PSQL.Username,
				cfg.Stores.PSQL.Password,
				cfg.Stores.PSQL.Database,
				store.WithDebug(cfg.Stores.PSQL.Debug),
			))
		default:
			return fmt.Errorf("unsupported storage: %v", s)
		}
	}
	if err != nil {
		return fmt.Errorf("create store client: %v", err)
	}
	defer multierr.AppendInvoke(&errReturned, multierr.Close(storage))

	// Migrations.
	if err = storage.Schema.Create(ctx); err != nil {
		return fmt.Errorf("migrate: %v", err)
	}

	// Repositories.
	db := store.NewDatabase(storage)

	chatsRepo, err := chatsrepo.New(chatsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create chats repo: %v", err)
	}

	jobsRepo, err := jobsrepo.New(jobsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create jobs repo: %v", err)
	}

	msgRepo, err := messagesrepo.New(messagesrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create message repo: %v", err)
	}

	problemsRepo, err := problemsrepo.New(problemsrepo.NewOptions(db))
	if err != nil {
		return fmt.Errorf("create problems repo: %v", err)
	}

	// Clients.
	kc, err := keycloakclient.New(keycloakclient.NewOptions(
		cfg.Clients.Keycloak.BasePath,
		cfg.Clients.Keycloak.Realm,
		cfg.Clients.Keycloak.ClientID,
		cfg.Clients.Keycloak.ClientSecret,
		keycloakclient.WithDebugMode(cfg.Clients.Keycloak.DebugMode),
	))
	if err != nil {
		return fmt.Errorf("create keycloak client: %v", err)
	}
	if cfg.IsProduction() && cfg.Clients.Keycloak.DebugMode {
		zap.L().Warn("keycloak client in the debug mode")
	}

	// Infrastructure Services.
	eventStream := inmemeventstream.New()
	defer multierr.AppendInvoke(&errReturned, multierr.Close(eventStream))

	msgProducer, err := msgproducer.New(msgproducer.NewOptions(
		msgproducer.NewKafkaWriter(
			cfg.Services.MsgProducer.Brokers,
			cfg.Services.MsgProducer.Topic,
			cfg.Services.MsgProducer.BatchSize,
		),
		msgproducer.WithEncryptKey(cfg.Services.MsgProducer.EncryptKey),
	))
	if err != nil {
		return fmt.Errorf("create message producer: %v", err)
	}
	defer multierr.AppendInvoke(&errReturned, multierr.Close(msgProducer))

	outBox, err := outbox.New(outbox.NewOptions(
		cfg.Services.Outbox.Workers,
		cfg.Services.Outbox.IdleTime,
		cfg.Services.Outbox.ReserveFor,
		jobsRepo,
		db,
	))
	if err != nil {
		return fmt.Errorf("create outbox service: %v", err)
	}

	managerPool := inmemmanagerpool.New()
	defer multierr.AppendInvoke(&errReturned, multierr.Close(managerPool))

	// Domain Services.
	managerLoad, err := managerload.New(managerload.NewOptions(
		cfg.Services.ManagerLoad.MaxProblemsAtSameTime,
		problemsRepo,
	))
	if err != nil {
		return fmt.Errorf("create manager load service: %v", err)
	}

	managerScheduler, err := managerscheduler.New(managerscheduler.NewOptions(
		cfg.Services.ManagerScheduler.Period,
		managerPool,
		msgRepo,
		outBox,
		problemsRepo,
		db,
		lg,
	))
	if err != nil {
		return fmt.Errorf("create manager scheduler: %v", err)
	}

	// Application Services.
	afcVerdictProcessor, err := afcverdictsprocessor.New(afcverdictsprocessor.NewOptions(
		cfg.Services.AFCVerdictsProcessor.Brokers,
		cfg.Services.AFCVerdictsProcessor.Consumers,
		cfg.Services.AFCVerdictsProcessor.ConsumerGroup,
		cfg.Services.AFCVerdictsProcessor.VerdictsTopic,
		afcverdictsprocessor.NewKafkaReader,
		afcverdictsprocessor.NewKafkaDLQWriter(
			cfg.Services.AFCVerdictsProcessor.Brokers,
			cfg.Services.AFCVerdictsProcessor.VerdictsDLQTopic,
		),
		db,
		msgRepo,
		outBox,
		afcverdictsprocessor.WithProcessBatchSize(cfg.Services.AFCVerdictsProcessor.BatchSize),
		afcverdictsprocessor.WithVerdictsSignKey(cfg.Services.AFCVerdictsProcessor.VerdictsSigningPublicKey),
	))
	if err != nil {
		return fmt.Errorf("create afc verdicts processor: %v", err)
	}

	// Application Services. Jobs.
	for _, j := range []outbox.Job{
		clientmessageblockedjob.Must(clientmessageblockedjob.NewOptions(eventStream, msgRepo)),
		clientmessagesentjob.Must(clientmessagesentjob.NewOptions(eventStream, msgRepo)),
		sendclientmessagejob.Must(sendclientmessagejob.NewOptions(eventStream, msgProducer, msgRepo)),
		managerassignedtoproblemjob.Must(managerassignedtoproblemjob.NewOptions(msgRepo, problemsRepo, managerLoad, eventStream)),
	} {
		outBox.MustRegisterJob(j)
	}

	// Servers.
	clientV1Swagger, err := clientv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("get client v1 swagger: %v", err)
	}

	srvClient, err := initServerClient(
		cfg.IsProduction(),
		cfg.Servers.Client.Addr,
		cfg.Servers.Client.AllowOrigins,
		clientV1Swagger,
		kc,
		cfg.Servers.Client.RequiredAccess.Resource,
		cfg.Servers.Client.RequiredAccess.Role,
		cfg.Servers.Client.SecWsProtocol,
		eventStream,
		outBox,
		db,
		chatsRepo,
		msgRepo,
		problemsRepo,
	)
	if err != nil {
		return fmt.Errorf("init client server: %v", err)
	}

	managerV1Swagger, err := managerv1.GetSwagger()
	if err != nil {
		return fmt.Errorf("get manager v1 swagger: %v", err)
	}

	srvManager, err := initServerManager(
		cfg.IsProduction(),
		cfg.Servers.Manager.Addr,
		cfg.Servers.Manager.AllowOrigins,
		managerV1Swagger,
		kc,
		cfg.Servers.Manager.RequiredAccess.Resource,
		cfg.Servers.Manager.RequiredAccess.Role,
		cfg.Servers.Manager.SecWsProtocol,
		eventStream,
		managerLoad,
		managerPool,
		chatsRepo,
	)
	if err != nil {
		return fmt.Errorf("init manager server: %v", err)
	}

	clientEventsSwagger, err := clientevents.GetSwagger()
	if err != nil {
		return fmt.Errorf("get client events swagger: %v", err)
	}

	managerEventsSwagger, err := managerevents.GetSwagger()
	if err != nil {
		return fmt.Errorf("get manager events swagger: %v", err)
	}

	srvDebug, err := serverdebug.New(serverdebug.NewOptions(
		cfg.Servers.Debug.Addr,
		clientV1Swagger,
		managerV1Swagger,
		clientEventsSwagger,
		managerEventsSwagger,
	))
	if err != nil {
		return fmt.Errorf("init debug server: %v", err)
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Run servers.
	eg.Go(func() error { return srvDebug.Run(ctx) })
	eg.Go(func() error { return srvClient.Run(ctx) })
	eg.Go(func() error { return srvManager.Run(ctx) })

	// Run services.
	eg.Go(func() error { return outBox.Run(ctx) })
	eg.Go(func() error { return afcVerdictProcessor.Run(ctx) })
	eg.Go(func() error { return managerScheduler.Run(ctx) })

	if err = eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("wait app stop: %v", err)
	}

	return nil
}
