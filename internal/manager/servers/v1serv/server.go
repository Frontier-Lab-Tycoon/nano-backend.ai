package v1serv

import (
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/manager/servers/v1serv/runserv"
	"github.com/seedspirit/nano-backend.ai/internal/manager/service"
)

// ServerArgs configures v1 API routes.
type ServerArgs struct {
	Services *service.Services
}

// WithSubServer registers v1 API routes below the given group.
func WithSubServer(g *echo.Group, args ServerArgs) error {
	v1Group := g.Group("/v1")
	if err := runserv.WithSubServer(v1Group, runserv.Args{
		Services: args.Services,
	}); err != nil {
		return err
	}
	return nil
}
