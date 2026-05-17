package spec

import (
	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
)

// Spec is the immutable, finalized input used to create a Run.
//
// A Spec is persisted independently of any Run it spawns, so the same Spec
// can be referenced by multiple Runs. The Spec captures what to run; the Run
// captures the act of running it.
type Spec struct {
	ID              uuid.UUID       `json:"id"`
	ProjectID       uuid.UUID       `json:"project_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	PresetRefs      preset.Refs     `json:"preset_refs,omitempty"`
	ModelOptions    ModelOptions    `json:"model_options"`
	DataOptions     DataOptions     `json:"data_options"`
	ResourceOptions ResourceOptions `json:"resource_options"`
	TrainingOptions TrainingOptions `json:"training_options"`
}

// New creates a finalized Spec with a fresh ID and the given identifying fields.
func New(projectID uuid.UUID, name string) Spec {
	return Spec{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
	}
}
