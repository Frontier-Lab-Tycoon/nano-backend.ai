package runserv

import (
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/manager/service"
)

// Args configures the run API routes.
type Args struct {
	Services *service.Services
}

// WithSubServer registers run API routes below the given group.
func WithSubServer(g *echo.Group, args Args) error {
	handler, err := newRunHandler(handlerArgs(args))
	if err != nil {
		return err
	}
	runGroup := g.Group("/runs")
	runGroup.GET("/:id/spec", handler.getSpec)
	return nil
}
