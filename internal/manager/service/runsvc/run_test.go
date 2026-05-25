package runsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

type stubBuilder struct {
	spec spec.Spec
	err  error
}

func (b *stubBuilder) Build(ctx context.Context, d *draft.Draft) (spec.Spec, error) {
	if b.err != nil {
		return spec.Spec{}, b.err
	}
	return b.spec, nil
}

type stubRunRepo struct {
	projectExistsErr error
	createRunErr     error
	created          bool
}

func (r *stubRunRepo) ProjectExists(ctx context.Context, projectID uuid.UUID) error {
	return r.projectExistsErr
}

func (r *stubRunRepo) CreateRun(ctx context.Context, s *spec.Spec, runRecord *run.Run) error {
	r.created = true
	return r.createRunErr
}

func (r *stubRunRepo) GetSpec(ctx context.Context, id uuid.UUID) (spec.Spec, error) {
	return spec.Spec{}, errordef.ErrNotFound
}

func (r *stubRunRepo) ListProjectRuns(ctx context.Context, projectID uuid.UUID, limit int) ([]run.Run, error) {
	return nil, nil
}

func TestSubmitReturnsQueuedRunOnSuccess(t *testing.T) {
	projectID := uuid.New()
	specID := uuid.New()
	builder := &stubBuilder{spec: spec.Spec{ID: specID, ProjectID: projectID}}
	repo := &stubRunRepo{}
	svc := &Service{repo: repo, specBuilder: builder}

	d := draft.Draft{ID: uuid.New(), ProjectID: projectID}
	got, err := svc.Submit(context.Background(), &d)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if got.SpecID == specID {
		t.Fatalf("got spec id %s; want a freshly assigned UUID different from stub processor's %s", got.SpecID, specID)
	}
	if got.Lifecycle.Status != run.Queued {
		t.Fatalf("got status %s, want queued", got.Lifecycle.Status)
	}
	if !repo.created {
		t.Fatal("repo.CreateRun not called")
	}
}

func TestSubmitReturnsNotFoundForMissingProject(t *testing.T) {
	builder := &stubBuilder{}
	repo := &stubRunRepo{projectExistsErr: errordef.ErrNotFound}
	svc := &Service{repo: repo, specBuilder: builder}

	_, err := svc.Submit(context.Background(), &draft.Draft{ID: uuid.New(), ProjectID: uuid.New()})
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
