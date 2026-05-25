package projectserv

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/dto/response"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

// defaultProjectRunsLimit is the Phase 0 list size before cursor pagination.
const defaultProjectRunsLimit = 20

type projectHandler struct {
	svc runService
}

type runService interface {
	ListProjectRuns(ctx context.Context, projectID uuid.UUID, limit int) ([]run.Run, error)
}

func newProjectHandler(args Args) (*projectHandler, error) {
	if args.Services == nil || args.Services.RunSvc == nil {
		return nil, errordef.Errorf(errordef.InvalidInput, "run service is required")
	}
	return &projectHandler{svc: args.Services.RunSvc}, nil
}

func (h *projectHandler) listRuns(c *echo.Context) error {
	ctx := c.Request().Context()
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		status, payload := errordef.Response(errordef.ErrInvalidInput, "check the project ID and retry", nil)
		return c.JSON(status, payload)
	}

	runs, err := h.svc.ListProjectRuns(ctx, projectID, defaultProjectRunsLimit)
	if err != nil {
		status, payload := errordef.Response(err, "retry later or contact an operator", nil)
		return c.JSON(status, payload)
	}

	data := response.ProjectRunsData{
		Runs:  response.NewRunSummaries(runs),
		Limit: defaultProjectRunsLimit,
	}
	return c.JSON(http.StatusOK, response.OK(data))
}
