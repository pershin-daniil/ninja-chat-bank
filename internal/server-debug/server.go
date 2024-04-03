package serverdebug

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
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
	addr      string      `option:"mandatory" validate:"required,hostname_port"`
	v1Swagger *openapi3.T `option:"mandatory" validate:"required"`
}

type Server struct {
	lg      *zap.Logger
	srv     *http.Server
	swagger *openapi3.T
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate options serverdebug: %v", err)
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
		swagger: opts.v1Swagger,
	}
	index := newIndexPage()

	e.GET("/version", s.version)
	index.addPage("/version", "Get build information")

	e.PUT("/log/level", echo.WrapHandler(logger.Level))

	{
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		pprofMux.HandleFunc("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
		pprofMux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
		pprofMux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
		pprofMux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
		pprofMux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
		pprofMux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)

		e.GET("/debug/pprof/*", echo.WrapHandler(pprofMux))
		index.addPage("/debug/pprof/", "Go std profiler")
		index.addPage("/debug/pprof/profile?seconds=30", "Take half-min profile")
	}

	e.GET("/debug/error", s.error)
	index.addPage("/debug/error", "Send Sentry error event")

	e.GET("/debug/log-levels", s.logLevels)
	index.addPage("/debug/log-levels", "Send all log levels messages")

	e.GET("/schema/client", s.schema)
	index.addPage("/schema/client", "Get client OpenAPI specification")

	e.GET("/", index.handler)
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return s.srv.Shutdown(ctx) //nolint:contextcheck // graceful shutdown with new context
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

func (s *Server) version(eCtx echo.Context) error {
	return eCtx.JSON(http.StatusOK, buildinfo.BuildInfo)
}

func (s *Server) logLevels(eCtx echo.Context) error {
	s.lg.Debug("🐞 DEBUG")
	s.lg.Info("ℹ️ INFO")
	s.lg.Warn("⚠️ WARN")
	s.lg.Error("❌ ERROR")

	return eCtx.String(http.StatusOK, "events sent")
}

func (s *Server) error(eCtx echo.Context) error {
	s.lg.Error("❌ ERROR")

	return eCtx.String(http.StatusOK, "event sent")
}

func (s *Server) schema(eCtx echo.Context) error {
	data, err := s.swagger.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal swagger: %v", err)
	}

	if err = eCtx.Blob(http.StatusOK, "application/json", data); err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}

	return nil
}
