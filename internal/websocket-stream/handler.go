package websocketstream

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/pershin-daniil/ninja-chat-bank/internal/middlewares"
	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

const (
	writeTimeout = time.Second
)

type eventStream interface {
	Subscribe(ctx context.Context, userID types.UserID) (<-chan eventstream.Event, error)
}

//go:generate options-gen -out-filename=handler_options.gen.go -from-struct=Options
type Options struct {
	pingPeriod time.Duration `default:"3s" validate:"omitempty,min=100ms,max=30s"`

	logger       *zap.Logger     `option:"mandatory" validate:"required"`
	eventStream  eventStream     `option:"mandatory" validate:"required"`
	eventAdapter EventAdapter    `option:"mandatory" validate:"required"`
	eventWriter  EventWriter     `option:"mandatory" validate:"required"`
	upgrader     Upgrader        `option:"mandatory" validate:"required"`
	shutdownCh   <-chan struct{} `option:"mandatory" validate:"required"`
}

type HTTPHandler struct {
	Options
}

func NewHTTPHandler(opts Options) (*HTTPHandler, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options websocketstream: %v", err)
	}

	return &HTTPHandler{Options: opts}, nil
}

func (h *HTTPHandler) Serve(eCtx echo.Context) error {
	ws, err := h.upgrader.Upgrade(eCtx.Response(), eCtx.Request(), nil)
	if err != nil {
		return fmt.Errorf("upgrade ws: %v", err)
	}

	ctx := eCtx.Request().Context()
	userID := middlewares.MustUserID(eCtx)

	events, err := h.eventStream.Subscribe(ctx, userID)
	if err != nil {
		return fmt.Errorf("subscribe on event stream: %v", err)
	}

	closer := newWsCloser(h.logger, ws)
	defer closer.Close(websocket.CloseNormalClosure)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return h.readLoop(ctx, ws) })

	eg.Go(func() error { return h.writeLoop(ctx, ws, events) })

	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case <-h.shutdownCh:
			closer.Close(websocket.CloseNormalClosure)
			return nil
		}
	})

	return eg.Wait()
}

// readLoop listen PONGs.
func (h *HTTPHandler) readLoop(_ context.Context, ws Websocket) error {
	pongDeadline := 2 * h.pingPeriod

	err := ws.SetReadDeadline(time.Now().Add(pongDeadline))
	if err != nil {
		return fmt.Errorf("set read deadline: %v", err)
	}

	ws.SetPongHandler(func(string) error {
		err = ws.SetReadDeadline(time.Now().Add(pongDeadline))
		if err != nil {
			h.logger.Debug("set read deadline", zap.Error(err))
		}

		h.logger.Debug("pong")

		return nil
	})

	for {
		_, _, err = ws.NextReader()
		if err != nil {
			return fmt.Errorf("read next reader: %v", err)
		}
	}
}

// writeLoop listen events and writes them into Websocket.
func (h *HTTPHandler) writeLoop(ctx context.Context, ws Websocket, events <-chan eventstream.Event) error {
	t := time.NewTicker(h.pingPeriod)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			err := h.writePing(ws)
			if err != nil {
				return fmt.Errorf("write ping: %v", err)
			}
		case event, ok := <-events:
			if !ok {
				h.logger.Warn("events channel closed")
				return nil
			}
			err := h.writeEvent(ws, event)
			if err != nil {
				return fmt.Errorf("write event: %v", err)
			}
		}
	}
}

func (h *HTTPHandler) writePing(ws Websocket) error {
	err := ws.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return fmt.Errorf("set write deadline: %v", err)
	}

	err = ws.WriteMessage(websocket.PingMessage, nil)
	if err != nil {
		return fmt.Errorf("write ping message: %v", err)
	}

	h.logger.Debug("ping")

	return nil
}

func (h *HTTPHandler) writeEvent(ws Websocket, event eventstream.Event) error {
	err := ws.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return fmt.Errorf("set write deadline: %v", err)
	}

	w, err := ws.NextWriter(websocket.TextMessage)
	if err != nil {
		return fmt.Errorf("get next writer: %v", err)
	}

	result, err := h.eventAdapter.Adapt(event)
	if err != nil {
		return fmt.Errorf("adapt event: %v", err)
	}

	err = h.eventWriter.Write(result, w)
	if err != nil {
		return fmt.Errorf("write event: %v", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("close writer: %v", err)
	}

	return nil
}
