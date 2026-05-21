package entity

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
)

// Spec is the database record shape for a spec row.
type Spec struct {
	ID              string                          `db:"id"`
	ProjectID       string                          `db:"project_id"`
	Name            string                          `db:"name"`
	Description     string                          `db:"description"`
	ModelOptions    jsonField[spec.ModelOptions]    `db:"model_options"`
	DataOptions     jsonField[spec.DataOptions]     `db:"data_options"`
	ResourceOptions jsonField[spec.ResourceOptions] `db:"resource_options"`
	TrainingOptions jsonField[spec.TrainingOptions] `db:"training_options"`
	CreatedAt       string                          `db:"created_at"`
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
		ID:              id,
		ProjectID:       projectID,
		Name:            s.Name,
		Description:     s.Description,
		PresetRefs:      s.PresetRefs,
		ModelOptions:    s.ModelOptions.Data,
		DataOptions:     s.DataOptions.Data,
		ResourceOptions: s.ResourceOptions.Data,
		TrainingOptions: s.TrainingOptions.Data,
	}

	return runSpec, nil
}
