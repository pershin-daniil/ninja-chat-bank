package logger

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"syscall"

	"github.com/TheZeroSlave/zapsentry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pershin-daniil/ninja-chat-bank/internal/buildinfo"
)

var Level zap.AtomicLevel

//go:generate options-gen -out-filename=logger_options.gen.go -from-struct=Options
type Options struct {
	level          string `option:"mandatory" validate:"required,oneof=debug info warn error"`
	productionMode bool
	sentryDNS      string
	env            string
}

func MustInit(opts Options) {
	if err := Init(opts); err != nil {
		panic(err)
	}
}

func Init(opts Options) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("failed to validate options: %v", err)
	}

	var err error
	Level, err = zap.ParseAtomicLevel(opts.level)
	if err != nil {
		return fmt.Errorf("failed to parse level: %v", err)
	}

	config := zap.NewProductionEncoderConfig()
	config.NameKey = "component"
	config.TimeKey = "T"
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if opts.productionMode {
		config.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(config)
	}

	cores := []zapcore.Core{
		zapcore.NewCore(encoder, os.Stdout, Level),
	}

	if opts.sentryDNS != "" {
		cfg := zapsentry.Configuration{
			Level: zapcore.WarnLevel,
		}

		client, e := NewSentryClient(opts.sentryDNS, opts.env, buildinfo.BuildInfo.Main.Version)
		if e != nil {
			return fmt.Errorf("failed to create new sentry client: %v", e)
		}

		core, e := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(client))
		if e != nil {
			return fmt.Errorf("failed to create new zapsentry core: %v", e)
		}

		cores = append(cores, core)
	}

	l := zap.New(zapcore.NewTee(cores...))
	zap.ReplaceGlobals(l)
	l.With(zapsentry.NewScope())

	return nil
}

func Sync() {
	if err := zap.L().Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) {
		stdlog.Printf("cannot sync logger: %v", err)
	}
}
