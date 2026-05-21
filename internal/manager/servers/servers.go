package servers

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/seedspirit/nano-backend.ai/internal/common/response"
	"github.com/seedspirit/nano-backend.ai/internal/manager/servers/v1serv"
	"github.com/seedspirit/nano-backend.ai/internal/manager/service"
)

// ServerArgs configures the manager HTTP server.
type ServerArgs struct {
	Addr     string
	Services *service.Services
}

// Server owns the HTTP listener and shutdown state.
type Server struct {
	httpServer *http.Server
	closed     atomic.Bool
}

// NewServer wires the Echo router into a net/http server.
func NewServer(args ServerArgs) (*Server, error) {
	e := echo.New()
	e.Binder = newBinder()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())
	e.GET("/health", health)
	if err := v1serv.WithSubServer(e.Group(""), v1serv.ServerArgs{
		Services: args.Services,
	}); err != nil {
		return nil, err
	}
	return &Server{
		httpServer: &http.Server{
			Addr:    args.Addr,
			Handler: e,
		},
		closed: atomic.Bool{},
	}, nil
}

func health(c *echo.Context) error {
	return c.JSON(http.StatusOK, response.OK(map[string]string{"state": "healthy"}))
}

// Start listens for HTTP requests until the server is stopped or fails.
func (server *Server) Start(ctx context.Context) error {
	if err := server.httpServer.ListenAndServe(); err != nil {
		if server.closed.Load() {
			slog.Info("server closed")
			return nil
		}
		slog.Error("server failed", "error", err)
		return err
	}
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (server *Server) Stop(ctx context.Context) error {
	server.closed.Store(true)
	return server.httpServer.Shutdown(ctx)
}
