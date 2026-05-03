package record

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
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
}

type comparableSpec struct {
	ProjectID       string              `json:"project_id"`
	Name            string              `json:"name"`
	Description     string              `json:"description,omitempty"`
	ModelOptions    run.ModelOptions    `json:"model_options"`
	DataOptions     run.DataOptions     `json:"data_options"`
	ResourceOptions run.ResourceOptions `json:"resource_options"`
	TrainingOptions run.TrainingOptions `json:"training_options"`
}

// NewSpec creates a spec record from the public spec type.
func NewSpec(spec *run.Spec) (Spec, error) {
	modelOptions, err := encoding.MarshalJSON(spec.ModelOptions)
	if err != nil {
		return Spec{}, err
	}
	dataOptions, err := encoding.MarshalJSON(spec.DataOptions)
	if err != nil {
		return Spec{}, err
	}
	resourceOptions, err := encoding.MarshalJSON(spec.ResourceOptions)
	if err != nil {
		return Spec{}, err
	}
	trainingOptions, err := encoding.MarshalJSON(spec.TrainingOptions)
	if err != nil {
		return Spec{}, err
	}

	return Spec{
		ID:              spec.ID.String(),
		ProjectID:       spec.ProjectID.String(),
		Name:            spec.Name,
		Description:     spec.Description,
		ModelOptions:    modelOptions,
		DataOptions:     dataOptions,
		ResourceOptions: resourceOptions,
		TrainingOptions: trainingOptions,
		CreatedAt:       encoding.FormatTime(time.Now()),
	}, nil
}

// ToSpec converts the database record into the public spec type.
func (s *Spec) ToSpec() (run.Spec, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return run.Spec{}, fmt.Errorf("parse spec id %q: %w", s.ID, err)
	}
	projectID, err := uuid.Parse(s.ProjectID)
	if err != nil {
		return run.Spec{}, fmt.Errorf("parse project id %q: %w", s.ProjectID, err)
	}

	spec := run.Spec{
		ID:          id,
		ProjectID:   projectID,
		Name:        s.Name,
		Description: s.Description,
	}
	if err := encoding.UnmarshalJSON(s.ModelOptions, &spec.ModelOptions); err != nil {
		return run.Spec{}, err
	}
	if err := encoding.UnmarshalJSON(s.DataOptions, &spec.DataOptions); err != nil {
		return run.Spec{}, err
	}
	if err := encoding.UnmarshalJSON(s.ResourceOptions, &spec.ResourceOptions); err != nil {
		return run.Spec{}, err
	}
	if err := encoding.UnmarshalJSON(s.TrainingOptions, &spec.TrainingOptions); err != nil {
		return run.Spec{}, err
	}

	return spec, nil
}

// ComparableJSON returns the stable JSON form used for idempotency comparison.
func (s *Spec) ComparableJSON() (string, error) {
	spec, err := s.ToSpec()
	if err != nil {
		return "", err
	}
	return ComparableSpecJSON(&spec)
}

// ComparableSpecJSON returns the stable JSON form used for idempotency comparison.
func ComparableSpecJSON(spec *run.Spec) (string, error) {
	fingerprint, err := json.Marshal(comparableSpec{
		ProjectID:       spec.ProjectID.String(),
		Name:            spec.Name,
		Description:     spec.Description,
		ModelOptions:    spec.ModelOptions,
		DataOptions:     spec.DataOptions,
		ResourceOptions: spec.ResourceOptions,
		TrainingOptions: spec.TrainingOptions,
	})
	if err != nil {
		return "", fmt.Errorf("compare spec %s: %w", spec.ID, err)
	}
	return string(fingerprint), nil
}
