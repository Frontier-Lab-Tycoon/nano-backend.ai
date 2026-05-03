package record

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
)

// Run is the database record shape for a run row.
type Run struct {
	ID               string         `db:"id"`
	ProjectID        string         `db:"project_id"`
	SpecID           string         `db:"spec_id"`
	IdempotencyKey   sql.NullString `db:"idempotency_key"`
	Status           string         `db:"status"`
	FailureReason    sql.NullString `db:"failure_reason"`
	ArtifactBasePath sql.NullString `db:"artifact_base_path"`
	CreatedAt        string         `db:"created_at"`
	StartedAt        sql.NullString `db:"started_at"`
	FinishedAt       sql.NullString `db:"finished_at"`
}

// NewRun creates a run record from the public run type.
func NewRun(rn *run.Run, projectID uuid.UUID) Run {
	return Run{
		ID:             rn.ID.String(),
		ProjectID:      projectID.String(),
		SpecID:         rn.SpecID.String(),
		IdempotencyKey: nullString(rn.IdempotencyKey),
		Status:         string(rn.Lifecycle.Status),
		FailureReason:  nullFailureReason(rn.Lifecycle.FailureReason),
		CreatedAt:      encoding.FormatTime(rn.Lifecycle.CreatedAt),
		StartedAt:      nullTime(rn.Lifecycle.StartedAt),
		FinishedAt:     nullTime(rn.Lifecycle.FinishedAt),
	}
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
	startedAt, err := parseNullTime(r.StartedAt)
	if err != nil {
		return run.Run{}, err
	}
	finishedAt, err := parseNullTime(r.FinishedAt)
	if err != nil {
		return run.Run{}, err
	}

	var key *string
	if r.IdempotencyKey.Valid {
		keyCopy := r.IdempotencyKey.String
		key = &keyCopy
	}
	var reason *run.FailureReason
	if r.FailureReason.Valid {
		reasonValue := run.FailureReason(r.FailureReason.String)
		reason = &reasonValue
	}

	return run.Run{
		ID:             id,
		SpecID:         specID,
		IdempotencyKey: key,
		Lifecycle: run.Lifecycle{
			Status:        run.Status(r.Status),
			FailureReason: reason,
			CreatedAt:     createdAt,
			StartedAt:     startedAt,
			FinishedAt:    finishedAt,
		},
	}, nil
}
