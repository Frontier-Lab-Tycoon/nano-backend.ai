package runsvc

import (
	"context"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/repository"
)

// Args configures the run service.
type Args struct {
	Repositories *repository.Repositories
}

// RunRepository is the persistence dependency required by the run service.
type RunRepository interface {
	GetSpec(ctx context.Context, id uuid.UUID) (spec.Spec, error)
}

// Service provides run use cases.
type Service struct {
	repo RunRepository
}

// NewService creates a run service.
func NewService(args Args) *Service {
	return &Service{
		repo: args.Repositories.Run,
	}
}

// GetSpec returns the spec associated with a run ID.
func (s *Service) GetSpec(ctx context.Context, id uuid.UUID) (spec.Spec, error) {
	return s.repo.GetSpec(ctx, id)
}
