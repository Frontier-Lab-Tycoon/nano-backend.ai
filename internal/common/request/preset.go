package request

import (
	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
)

// PresetRefsReq selects optional preset categories by ID in a submitted request.
type PresetRefsReq struct {
	Trainer  *uuid.UUID `json:"trainer,omitempty"`
	Resource *uuid.UUID `json:"resource,omitempty"`
	Output   *uuid.UUID `json:"output,omitempty"`
}

func presetRefsToDraft(req PresetRefsReq) preset.Refs {
	return preset.Refs{
		Trainer:  req.Trainer,
		Resource: req.Resource,
		Output:   req.Output,
	}
}
