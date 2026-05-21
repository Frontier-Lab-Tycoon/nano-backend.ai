package draft

import (
	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
)

// Draft is the persisted or processor-facing input used to create a finalized Spec.
type Draft struct {
	ID              uuid.UUID          `json:"id"`
	ProjectID       uuid.UUID          `json:"project_id"`
	Name            string             `json:"name"`
	Description     string             `json:"description,omitempty"`
	PresetRefs      preset.Refs        `json:"preset_refs,omitempty"`
	ModelOptions    ModelOptionsReq    `json:"model_options"`
	DataOptions     DataOptionsReq     `json:"data_options"`
	ResourceOptions ResourceOptionsReq `json:"resource_options"`
	TrainingOptions TrainingOptionsReq `json:"training_options"`
}
