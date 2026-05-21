package entity

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
)

func TestRunToRunMapsStoredFields(t *testing.T) {
	runID := uuid.New()
	specID := uuid.New()
	idempotencyKey := "submit-1"
	failureReason := "runtime_error"
	row := Run{
		ID:             runID.String(),
		SpecID:         specID.String(),
		IdempotencyKey: sql.NullString{String: idempotencyKey, Valid: true},
		Status:         string(run.Failed),
		FailureReason:  sql.NullString{String: failureReason, Valid: true},
		CreatedAt:      "2026-05-21T00:00:00Z",
		StartedAt:      sql.NullString{String: "2026-05-21T00:01:00Z", Valid: true},
		FinishedAt:     sql.NullString{String: "2026-05-21T00:02:00Z", Valid: true},
	}

	got, err := row.ToRun()
	if err != nil {
		t.Fatalf("to run: %v", err)
	}
	if got.ID != runID {
		t.Fatalf("got run id %s, want %s", got.ID, runID)
	}
	if got.SpecID != specID {
		t.Fatalf("got spec id %s, want %s", got.SpecID, specID)
	}
	if got.IdempotencyKey == nil || *got.IdempotencyKey != idempotencyKey {
		t.Fatalf("got idempotency key %v, want %s", got.IdempotencyKey, idempotencyKey)
	}
	if got.Lifecycle.Status != run.Failed {
		t.Fatalf("got status %s, want %s", got.Lifecycle.Status, run.Failed)
	}
	if got.Lifecycle.FailureReason == nil || string(*got.Lifecycle.FailureReason) != failureReason {
		t.Fatalf("got failure reason %v, want %s", got.Lifecycle.FailureReason, failureReason)
	}
	if got.Lifecycle.StartedAt == nil {
		t.Fatal("got nil started_at")
	}
	if got.Lifecycle.FinishedAt == nil {
		t.Fatal("got nil finished_at")
	}
}

func TestRunToRunReturnsErrorForInvalidRunID(t *testing.T) {
	row := Run{
		ID:        "not-a-uuid",
		SpecID:    uuid.New().String(),
		Status:    string(run.Queued),
		CreatedAt: "2026-05-21T00:00:00Z",
	}

	if _, err := row.ToRun(); err == nil {
		t.Fatal("got nil error, want invalid run ID error")
	}
}

func TestRunToRunReturnsErrorForInvalidTimestamp(t *testing.T) {
	row := Run{
		ID:        uuid.New().String(),
		SpecID:    uuid.New().String(),
		Status:    string(run.Queued),
		CreatedAt: "not-a-time",
	}

	if _, err := row.ToRun(); err == nil {
		t.Fatal("got nil error, want invalid timestamp error")
	}
}
