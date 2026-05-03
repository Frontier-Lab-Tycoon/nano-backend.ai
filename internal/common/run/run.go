// Package run defines the Run lifecycle types: Spec (input contract), Run
// (execution instance), Status (state machine), resource/option groups, and
// the artifact index produced by a completed Run.
package run

import (
	"time"

	"github.com/google/uuid"
)

// Run represents a single execution instance of a Spec.
//
// A Run owns its identity and lifecycle metadata (timestamps, status) and
// references the Spec it was created from. The same Spec may spawn multiple
// Runs (e.g., reproducibility re-runs), each distinguished by its own id and
// optional client-provided idempotency key.
type Run struct {
	ID             uuid.UUID `json:"id"`
	SpecID         uuid.UUID `json:"spec_id"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty"`
	Lifecycle      Lifecycle `json:"lifecycle"`
}

// NewRun creates a Run referencing the given Spec, in Queued status with the
// current time as CreatedAt. IdempotencyKey, StartedAt, and FinishedAt are
// set later by the application and runtime layers.
func NewRun(specID uuid.UUID) Run {
	return Run{
		ID:        uuid.New(),
		SpecID:    specID,
		Lifecycle: NewLifecycle(time.Now()),
	}
}

// Transition applies a lifecycle transition to the Run.
func (r *Run) Transition(transition Transition, at time.Time) error {
	return r.Lifecycle.Transition(transition, at)
}
