package logger

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var AtomicLevel zap.AtomicLevel

//go:generate options-gen -out-filename=logger_options.gen.go -from-struct=Options
type Options struct {
	level          string `option:"mandatory" validate:"required,oneof=debug info warn error"`
	productionMode bool
}

func MustInit(opts Options) {
	if err := Init(opts); err != nil {
		panic(err)
	}
}

func Init(opts Options) error {
	err := opts.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate options: %v", err)
	}

	AtomicLevel, err = zap.ParseAtomicLevel(opts.level)
	if err != nil {
		return fmt.Errorf("failed to parse level: %w", err)
	}

	config := zap.NewProductionEncoderConfig()
	config.NameKey = "component"
	config.TimeKey = "T"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	enc := zapcore.NewConsoleEncoder(config)

	if opts.productionMode {
		AtomicLevel.SetLevel(zapcore.InfoLevel)
		config.EncodeLevel = zapcore.CapitalLevelEncoder
		enc = zapcore.NewJSONEncoder(config)
	}

	ws := zapcore.Lock(os.Stdout)

	cores := []zapcore.Core{
		zapcore.NewCore(enc, ws, AtomicLevel),
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

func ChangeLevel(level string) error {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return err
	}
	AtomicLevel.SetLevel(lvl)
	return nil
}
