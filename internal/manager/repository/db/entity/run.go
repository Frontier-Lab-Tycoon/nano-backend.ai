package entity

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
)

// Run is the database record shape for a run row.
type Run struct {
	ID             string         `db:"id"`
	SpecID         string         `db:"spec_id"`
	IdempotencyKey sql.NullString `db:"idempotency_key"`
	Status         string         `db:"status"`
	FailureReason  sql.NullString `db:"failure_reason"`
	CreatedAt      string         `db:"created_at"`
	StartedAt      sql.NullString `db:"started_at"`
	FinishedAt     sql.NullString `db:"finished_at"`
}

// ToRun converts the database record into the public run type.
func (r *Run) ToRun() (run.Run, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return run.Run{}, fmt.Errorf("parse run id %q: %w", r.ID, err)
	}
	specID, err := uuid.Parse(r.SpecID)
	if err != nil {
		return run.Run{}, fmt.Errorf("parse spec id %q: %w", r.SpecID, err)
	}
	createdAt, err := encoding.ParseTime(r.CreatedAt)
	if err != nil {
		return run.Run{}, err
	}

	lifecycle := run.Lifecycle{
		Status:    run.Status(r.Status),
		CreatedAt: createdAt,
	}
	if r.FailureReason.Valid {
		reason := run.FailureReason(r.FailureReason.String)
		lifecycle.FailureReason = &reason
	}
	if r.StartedAt.Valid {
		startedAt, err := encoding.ParseTime(r.StartedAt.String)
		if err != nil {
			return run.Run{}, err
		}
		lifecycle.StartedAt = &startedAt
	}
	if r.FinishedAt.Valid {
		finishedAt, err := encoding.ParseTime(r.FinishedAt.String)
		if err != nil {
			return run.Run{}, err
		}
		lifecycle.FinishedAt = &finishedAt
	}

	result := run.Run{
		ID:        id,
		SpecID:    specID,
		Lifecycle: lifecycle,
	}
	if r.IdempotencyKey.Valid {
		key := r.IdempotencyKey.String
		result.IdempotencyKey = &key
	}
	return result, nil
}
