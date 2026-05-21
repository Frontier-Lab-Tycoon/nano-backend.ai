package projectserv

import (
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/manager/service"
)

// Args configures the project API routes.
type Args struct {
	Services *service.Services
}

// WithSubServer registers project API routes below the given group.
func WithSubServer(g *echo.Group, args Args) error {
	handler, err := newProjectHandler(args)
	if err != nil {
		return err
	}
	projectGroup := g.Group("/projects")
	projectGroup.GET("/:id/runs", handler.listRuns)
	return nil
}
