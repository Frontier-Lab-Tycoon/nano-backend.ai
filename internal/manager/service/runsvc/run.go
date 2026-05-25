package runsvc

import (
	"context"

	"github.com/google/uuid"

	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	"github.com/seedspirit/nano-backend.ai/internal/manager/repository"
)

// Args configures the run service.
type Args struct {
	Repositories *repository.Repositories
	SpecBuilder  SpecBuilder
}

// RunRepository is the persistence dependency required by the run service.
type RunRepository interface {
	GetSpec(ctx context.Context, id uuid.UUID) (spec.Spec, error)
	ListProjectRuns(ctx context.Context, projectID uuid.UUID, limit int) ([]run.Run, error)
	ProjectExists(ctx context.Context, projectID uuid.UUID) error
	CreateRun(ctx context.Context, spec *spec.Spec, run *run.Run) error
}

// SpecBuilder finalizes a submitted draft into an immutable spec.
type SpecBuilder interface {
	Build(ctx context.Context, d *draft.Draft) (spec.Spec, error)
}

// Service provides run use cases.
type Service struct {
	repo        RunRepository
	specBuilder SpecBuilder
}

// NewService creates a run service.
func NewService(args Args) *Service {
	return &Service{
		repo:        args.Repositories.Run,
		specBuilder: args.SpecBuilder,
	}
}

// GetSpec returns the spec associated with a run ID.
func (s *Service) GetSpec(ctx context.Context, id uuid.UUID) (spec.Spec, error) {
	return s.repo.GetSpec(ctx, id)
}

// ListProjectRuns returns the most recent runs associated with a project.
func (s *Service) ListProjectRuns(ctx context.Context, projectID uuid.UUID, limit int) ([]run.Run, error) {
	return s.repo.ListProjectRuns(ctx, projectID, limit)
}

// Submit validates a draft, finalizes a spec, and persists spec + queued run record.
func (s *Service) Submit(ctx context.Context, runDraft *draft.Draft) (run.Run, error) {
	if runDraft == nil {
		return run.Run{}, errordef.Errorf(errordef.InvalidInput, "draft is nil")
	}
	if err := s.repo.ProjectExists(ctx, runDraft.ProjectID); err != nil {
		return run.Run{}, err
	}

	built, err := s.specBuilder.Build(ctx, runDraft)
	if err != nil {
		return run.Run{}, err
	}

	built.ID = uuid.New()
	built.ProjectID = runDraft.ProjectID

	newRun := run.NewWithSpec(built.ID, runDraft.ProjectID)
	if err := s.repo.CreateRun(ctx, &built, &newRun); err != nil {
		return run.Run{}, err
	}
	return newRun, nil
}
