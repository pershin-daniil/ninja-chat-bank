package server_debug

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/pershin-daniil/ninja-chat-bank/internal/buildinfo"
	"github.com/pershin-daniil/ninja-chat-bank/internal/logger"
)

const (
	readHeaderTimeout = time.Second
	shutdownTimeout   = 3 * time.Second
)

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options
type Options struct {
	addr string `option:"mandatory" validate:"required,hostname_port"`
}

type Server struct {
	lg  *zap.Logger
	srv *http.Server
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate options: %v", err)
	}

	lg := zap.L().Named("server-debug")

	e := echo.New()
	e.Use(middleware.Recover())

	s := &Server{
		lg: lg,
		srv: &http.Server{
			Addr:              opts.addr,
			Handler:           e,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}
	index := newIndexPage()
	index.addPage("/version", "Get build information")
	index.addPage("/debug/pprof/", "Go std profiler")
	index.addPage("/debug/pprof/profile?seconds=30", "Take half-min profile")

	e.GET("/version", s.Version)
	e.PUT("/log/level", s.LogLevel)
	e.GET("/debug/pprof/", s.IndexHandler())
	e.GET("/debug/pprof/allocs", s.AllocHandler())
	e.GET("/debug/pprof/block", s.BlockHandler())
	e.GET("/debug/pprof/cmdline", s.CmdlineHandler())
	e.GET("/debug/pprof/goroutine", s.GoroutineHandler())
	e.GET("/debug/pprof/heap", s.HeapHandler())
	e.GET("/debug/pprof/mutex", s.MutexHandler())
	e.GET("/debug/pprof/profile", s.ProfileHandler())
	e.GET("/debug/pprof/threadcreate", s.ThreadCreateHandler())
	e.GET("/debug/pprof/trace", s.TraceHandler())

	e.GET("/", index.handler)
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return s.srv.Shutdown(ctx)
	})

	eg.Go(func() error {
		s.lg.Info("listen and serve", zap.String("addr", s.srv.Addr))

		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %v", err)
		}
		return nil
	})

	return eg.Wait()
}

func (s *Server) Version(eCtx echo.Context) error {
	if err := eCtx.JSON(http.StatusOK, buildinfo.BuildInfo); err != nil {
		return fmt.Errorf("sending version: %w", err)
	}
	return nil
}

func (s *Server) LogLevel(ectx echo.Context) error {
	level := ectx.FormValue("level")
	if err := logger.ChangeLevel(level); err != nil {
		return fmt.Errorf("failed to change level %s: %w", level, err)
	}
	s.lg.Info(fmt.Sprintf("log level changed to %s", level))
	return nil
}
