package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
)

func TestNewRunSummariesMapsRunData(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	specID := uuid.New()
	createdAt := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)

	got := NewRunSummaries(projectID, []run.Run{{
		ID:     runID,
		SpecID: specID,
		Lifecycle: run.Lifecycle{
			Status:    run.Queued,
			CreatedAt: createdAt,
		},
	}})

	if len(got) != 1 {
		t.Fatalf("got %d summaries, want 1", len(got))
	}
	if got[0].ID != runID {
		t.Fatalf("got run id %s, want %s", got[0].ID, runID)
	}
	if got[0].ProjectID != projectID {
		t.Fatalf("got project id %s, want %s", got[0].ProjectID, projectID)
	}
	if got[0].SpecID != specID {
		t.Fatalf("got spec id %s, want %s", got[0].SpecID, specID)
	}
	if got[0].Status != run.Queued {
		t.Fatalf("got status %s, want %s", got[0].Status, run.Queued)
	}
	if !got[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("got created_at %s, want %s", got[0].CreatedAt, createdAt)
	}
}

func TestNewRunSummariesReturnsEmptySlice(t *testing.T) {
	got := NewRunSummaries(uuid.New(), nil)

	if got == nil {
		t.Fatal("got nil summaries, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("got %d summaries, want 0", len(got))
	}
}
