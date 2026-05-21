package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
)

// RunRepository exposes run data needed by manager services.
type RunRepository interface {
	GetSpec(ctx context.Context, id uuid.UUID) (spec.Spec, error)
	Close() error
}
