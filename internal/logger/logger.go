package logger

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"syscall"

	"github.com/tchap/zapext/v2/zapsentry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pershin-daniil/ninja-chat-bank/internal/buildinfo"
)

const sentryLvl = zapcore.WarnLevel

var Level zap.AtomicLevel

//go:generate options-gen -out-filename=logger_options.gen.go -from-struct=Options
type Options struct {
	level          string `option:"mandatory" validate:"required,oneof=debug info warn error"`
	productionMode bool
	sentryDSN      string `validate:"omitempty,url"`
	sentryEnv      string `validate:"omitempty,oneof=dev stage prod"`
}

func MustInit(opts Options) {
	if err := Init(opts); err != nil {
		panic(err)
	}
}

func Init(opts Options) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("failed to validate options logger: %v", err)
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

	if dsn := opts.sentryDSN; dsn != "" {
		env := "dev"
		if opts.productionMode {
			env = "prod"
		}

		sentryClient, e := NewSentryClient(dsn, env, buildinfo.Version())
		if e != nil {
			return fmt.Errorf("new sentry client: %v", e)
		}

		cores = append(cores, zapsentry.NewCore(sentryLvl, sentryClient))
	}

	l := zap.New(zapcore.NewTee(cores...))
	zap.ReplaceGlobals(l)

	return nil
}

func Sync() {
	if err := zap.L().Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) {
		stdlog.Printf("cannot sync logger: %v", err)
	}
}
