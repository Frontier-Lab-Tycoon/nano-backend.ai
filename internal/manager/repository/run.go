package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/project"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/spec"
)

// RunRepository persists projects, specs, runs, and artifact indexes.
type RunRepository interface {
	CreateProject(ctx context.Context, p project.Project) error
	CreateSpec(ctx context.Context, runSpec *spec.Spec) error
	CreateRun(ctx context.Context, rn *run.Run, projectID uuid.UUID) error
	SubmitRun(ctx context.Context, runSpec *spec.Spec, idempotencyKey *string) (run.Run, error)
	GetRun(ctx context.Context, id uuid.UUID) (run.Run, error)
	ListRuns(ctx context.Context, projectID uuid.UUID) ([]run.Run, error)
	UpdateLifecycle(ctx context.Context, id uuid.UUID, lifecycle run.Lifecycle) error
	SaveArtifactIndex(ctx context.Context, index run.ArtifactIndex) error
	GetArtifactIndex(ctx context.Context, runID uuid.UUID) (run.ArtifactIndex, error)
	Close() error
}
