package record

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/spec"
)

// Spec is the database record shape for a spec row.
type Spec struct {
	ID              string `db:"id"`
	ProjectID       string `db:"project_id"`
	Name            string `db:"name"`
	Description     string `db:"description"`
	ModelOptions    string `db:"model_options"`
	DataOptions     string `db:"data_options"`
	ResourceOptions string `db:"resource_options"`
	TrainingOptions string `db:"training_options"`
	CreatedAt       string `db:"created_at"`
	PresetRefs      preset.Refs
}

// ToSpec converts the database record into the public spec type.
func (s *Spec) ToSpec() (spec.Spec, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return spec.Spec{}, fmt.Errorf("parse spec id %q: %w", s.ID, err)
	}
	projectID, err := uuid.Parse(s.ProjectID)
	if err != nil {
		return spec.Spec{}, fmt.Errorf("parse project id %q: %w", s.ProjectID, err)
	}

	runSpec := spec.Spec{
		ID:          id,
		ProjectID:   projectID,
		Name:        s.Name,
		Description: s.Description,
		PresetRefs:  s.PresetRefs,
	}
	if err := encoding.UnmarshalJSON(s.ModelOptions, &runSpec.ModelOptions); err != nil {
		return spec.Spec{}, err
	}
	if err := encoding.UnmarshalJSON(s.DataOptions, &runSpec.DataOptions); err != nil {
		return spec.Spec{}, err
	}
	if err := encoding.UnmarshalJSON(s.ResourceOptions, &runSpec.ResourceOptions); err != nil {
		return spec.Spec{}, err
	}
	if err := encoding.UnmarshalJSON(s.TrainingOptions, &runSpec.TrainingOptions); err != nil {
		return spec.Spec{}, err
	}

	return runSpec, nil
}
