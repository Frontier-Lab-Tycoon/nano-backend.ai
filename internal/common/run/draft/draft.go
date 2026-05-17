package draft

import (
	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
)

// Req is the user-submitted input used to create a Draft.
//
// Req can reference presets and user-supplied option requests, but it does not
// carry identity. Identity is assigned after the request is accepted.
type Req struct {
	ProjectID       uuid.UUID          `json:"project_id"`
	Name            string             `json:"name"`
	Description     string             `json:"description,omitempty"`
	PresetRefs      preset.Refs        `json:"preset_refs,omitempty"`
	ModelOptions    ModelOptionsReq    `json:"model_options"`
	DataOptions     DataOptionsReq     `json:"data_options"`
	ResourceOptions ResourceOptionsReq `json:"resource_options"`
	TrainingOptions TrainingOptionsReq `json:"training_options"`
}

// Draft is the persisted or processor-facing input used to create a finalized Spec.
type Draft struct {
	ID uuid.UUID `json:"id"`
	Req
}

// NewReq creates a Req with the given identifying fields.
func NewReq(projectID uuid.UUID, name string) Req {
	return Req{
		ProjectID: projectID,
		Name:      name,
	}
}

// New creates a Draft with a fresh ID and the given identifying fields.
func New(projectID uuid.UUID, name string) Draft {
	return Draft{
		ID:  uuid.New(),
		Req: NewReq(projectID, name),
	}
}

// FromReq assigns identity to a Req.
func FromReq(id uuid.UUID, req *Req) Draft {
	if req == nil {
		return Draft{ID: id}
	}
	return Draft{ID: id, Req: *req}
}
