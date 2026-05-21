package runserv

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/common/response"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	"github.com/seedspirit/nano-backend.ai/internal/manager/service"
)

type handlerArgs struct {
	Services *service.Services
}

type runHandler struct {
	svc runService
}

type runService interface {
	GetSpec(ctx context.Context, runID uuid.UUID) (spec.Spec, error)
}

func newRunHandler(args handlerArgs) (*runHandler, error) {
	if args.Services == nil || args.Services.RunSvc == nil {
		return nil, errordef.Errorf(errordef.InvalidInput, "run service is required")
	}
	return &runHandler{svc: args.Services.RunSvc}, nil
}

func (h *runHandler) getSpec(c *echo.Context) error {
	ctx := c.Request().Context()
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		status, payload := errordef.Response(errordef.ErrInvalidRunID, "check the run ID and retry", nil)
		return c.JSON(status, payload)
	}
	runSpec, err := h.svc.GetSpec(ctx, runID)
	if err != nil {
		status, payload := errordef.Response(err, "retry later or contact an operator", nil)
		return c.JSON(status, payload)
	}
	return c.JSON(http.StatusOK, response.OK(runSpec))
}
