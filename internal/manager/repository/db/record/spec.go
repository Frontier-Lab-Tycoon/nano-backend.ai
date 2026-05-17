package record

import (
	"encoding/json"
	"fmt"
	"time"

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

type comparableSpec struct {
	ProjectID       string               `json:"project_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	PresetRefs      preset.Refs          `json:"preset_refs"`
	ModelOptions    spec.ModelOptions    `json:"model_options"`
	DataOptions     spec.DataOptions     `json:"data_options"`
	ResourceOptions spec.ResourceOptions `json:"resource_options"`
	TrainingOptions spec.TrainingOptions `json:"training_options"`
}

// NewSpec creates a spec record from the public spec type.
func NewSpec(runSpec *spec.Spec) (Spec, error) {
	modelOptions, err := encoding.MarshalJSON(runSpec.ModelOptions)
	if err != nil {
		return Spec{}, err
	}
	dataOptions, err := encoding.MarshalJSON(runSpec.DataOptions)
	if err != nil {
		return Spec{}, err
	}
	resourceOptions, err := encoding.MarshalJSON(runSpec.ResourceOptions)
	if err != nil {
		return Spec{}, err
	}
	trainingOptions, err := encoding.MarshalJSON(runSpec.TrainingOptions)
	if err != nil {
		return Spec{}, err
	}

	return Spec{
		ID:              runSpec.ID.String(),
		ProjectID:       runSpec.ProjectID.String(),
		Name:            runSpec.Name,
		Description:     runSpec.Description,
		ModelOptions:    modelOptions,
		DataOptions:     dataOptions,
		ResourceOptions: resourceOptions,
		TrainingOptions: trainingOptions,
		CreatedAt:       encoding.FormatTime(time.Now()),
	}, nil
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

// ComparableJSON returns the stable JSON form used for idempotency comparison.
func (s *Spec) ComparableJSON() (string, error) {
	runSpec, err := s.ToSpec()
	if err != nil {
		return "", err
	}
	return ComparableSpecJSON(&runSpec)
}

// ComparableSpecJSON returns the stable JSON form used for idempotency comparison.
func ComparableSpecJSON(runSpec *spec.Spec) (string, error) {
	fingerprint, err := json.Marshal(comparableSpec{
		ProjectID:       runSpec.ProjectID.String(),
		Name:            runSpec.Name,
		Description:     runSpec.Description,
		PresetRefs:      runSpec.PresetRefs,
		ModelOptions:    runSpec.ModelOptions,
		DataOptions:     runSpec.DataOptions,
		ResourceOptions: runSpec.ResourceOptions,
		TrainingOptions: runSpec.TrainingOptions,
	})
	if err != nil {
		return "", fmt.Errorf("compare spec %s: %w", runSpec.ID, err)
	}
	return string(fingerprint), nil
}
