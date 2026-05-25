// Package run defines Run lifecycle types: Run, Status, primitive resource
// shapes, and the artifact index produced by a completed Run.
package run

import (
	"time"

	"github.com/google/uuid"
)

// Run represents a single execution instance of a Spec.
//
// A Run owns its identity and lifecycle metadata (timestamps, status) and
// references the Project + Spec it was created from. The same Spec may spawn
// multiple Runs (e.g., reproducibility re-runs), each distinguished by its
// own id and optional client-provided idempotency key.
type Run struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"project_id"`
	SpecID         uuid.UUID `json:"spec_id"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty"`
	Lifecycle      Lifecycle `json:"lifecycle"`
}

// NewWithSpec creates a Run for the given project and spec in Queued status
func NewWithSpec(specID, projectID uuid.UUID) Run {
	return Run{
		ID:        uuid.New(),
		ProjectID: projectID,
		SpecID:    specID,
		Lifecycle: NewLifecycle(time.Now()),
	}
}

// Transition applies a lifecycle transition to the Run.
func (r *Run) Transition(transition Transition, at time.Time) error {
	return r.Lifecycle.Transition(transition, at)
}
