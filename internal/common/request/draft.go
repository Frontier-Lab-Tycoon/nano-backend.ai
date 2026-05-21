// Package request holds user-submitted input shapes for the API boundary.
//
// Request types carry no identity and no domain invariants. They are decoded
// from transport, then converted into identified domain objects before they
// enter application workflow.
package request

import (
	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/draft"
)

// RunDraftReq is the user-submitted input used to create a run draft.
type RunDraftReq struct {
	ProjectID       uuid.UUID          `json:"project_id"`
	Name            string             `json:"name"`
	Description     string             `json:"description,omitempty"`
	PresetRefs      PresetRefsReq      `json:"preset_refs,omitempty"`
	ModelOptions    ModelOptionsReq    `json:"model_options"`
	DataOptions     DataOptionsReq     `json:"data_options"`
	ResourceOptions ResourceOptionsReq `json:"resource_options"`
	TrainingOptions TrainingOptionsReq `json:"training_options"`
}

// ToDraft assigns identity to a submitted request and converts it into a Draft.
func (req *RunDraftReq) ToDraft(id uuid.UUID) draft.Draft {
	if req == nil {
		return draft.Draft{ID: id}
	}
	return draft.Draft{
		ID:              id,
		ProjectID:       req.ProjectID,
		Name:            req.Name,
		Description:     req.Description,
		PresetRefs:      presetRefsToDraft(req.PresetRefs),
		ModelOptions:    modelOptionsToDraft(req.ModelOptions),
		DataOptions:     dataOptionsToDraft(req.DataOptions),
		ResourceOptions: resourceOptionsToDraft(req.ResourceOptions),
		TrainingOptions: trainingOptionsToDraft(req.TrainingOptions),
	}
}
