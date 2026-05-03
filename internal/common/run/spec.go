package run

import "github.com/google/uuid"

// Spec is the input contract submitted to create a Run.
//
// A Spec is persisted independently of any Run it spawns, so the same Spec
// can be referenced by multiple Runs (reproducibility re-runs, retries with
// different idempotency keys, etc.). The Spec captures *what* to run; the
// Run captures the act of running it.
type Spec struct {
	ID              uuid.UUID       `json:"id"`
	ProjectID       uuid.UUID       `json:"project_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	ModelOptions    ModelOptions    `json:"model_options"`
	DataOptions     DataOptions     `json:"data_options"`
	ResourceOptions ResourceOptions `json:"resource_options"`
	TrainingOptions TrainingOptions `json:"training_options"`
}

// NewSpec creates a Spec with a fresh ID and the given identifying fields.
//
// The option groups (ModelOptions, DataOptions, ResourceOptions,
// TrainingOptions) are zero-valued at creation; populate them via field
// assignment before submitting or persisting.
func NewSpec(projectID uuid.UUID, name string) Spec {
	return Spec{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
	}
}
