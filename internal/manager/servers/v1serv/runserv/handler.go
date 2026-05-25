package runserv

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/common/dto/request"
	"github.com/seedspirit/nano-backend.ai/internal/common/dto/response"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

type runHandler struct {
	svc runService
}

type runService interface {
	GetSpec(ctx context.Context, runID uuid.UUID) (spec.Spec, error)
	Submit(ctx context.Context, runDraft *draft.Draft) (run.Run, error)
}

func newRunHandler(args Args) (*runHandler, error) {
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

func (h *runHandler) submit(c *echo.Context) error {
	ctx := c.Request().Context()
	var req request.RunDraftReq
	if err := c.Bind(&req); err != nil {
		status, payload := errordef.Response(
			errordef.Errorf(errordef.ValidationError, "invalid request body: %s", err.Error()),
			"check the request body shape and retry",
			nil,
		)
		return c.JSON(status, payload)
	}

	runDraft := req.ToDraft(uuid.New())

	created, err := h.svc.Submit(ctx, &runDraft)
	if err != nil {
		hint := "retry later or contact an operator"
		if errors.Is(err, errordef.ErrNotFound) {
			hint = "check project_id and retry"
		}
		status, payload := errordef.Response(err, hint, nil)
		return c.JSON(status, payload)
	}

	summary := response.NewRunSummary(&created)
	return c.JSON(http.StatusCreated, response.Success(http.StatusCreated, summary))
}
