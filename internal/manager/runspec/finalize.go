package runspec

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/preset"
)

// FinalizedRunSpec is the immutable structured input produced after RunSpec processing.
type FinalizedRunSpec struct {
	SpecID          uuid.UUID               `json:"spec_id"`
	ProjectID       uuid.UUID               `json:"project_id"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description,omitempty"`
	ModelOptions    run.ModelOptions        `json:"model_options"`
	DataOptions     run.DataOptions         `json:"data_options"`
	ResourceOptions run.ResourceOptions     `json:"resource_options"`
	TrainerPresetID preset.ID               `json:"trainer_preset_id"`
	TrainingConfig  FinalizedTrainingConfig `json:"training_config"`
}

// FinalizedTrainingConfig is the structured training config that runtime adapters may materialize.
type FinalizedTrainingConfig struct {
	Values map[string]any `json:"values"`
}

// FinalizeRunSpec applies preset defaults and user parameters to create finalized data.
func FinalizeRunSpec(spec *run.Spec, trainerPreset preset.Preset) FinalizedRunSpec {
	values := trainerPreset.Defaults()
	if values == nil {
		values = make(map[string]any)
	}
	for key, value := range spec.TrainingOptions.Parameters {
		values[key] = cloneFinalizedValue(value)
	}

	return FinalizedRunSpec{
		SpecID:          spec.ID,
		ProjectID:       spec.ProjectID,
		Name:            spec.Name,
		Description:     spec.Description,
		ModelOptions:    spec.ModelOptions,
		DataOptions:     spec.DataOptions,
		ResourceOptions: spec.ResourceOptions,
		TrainerPresetID: trainerPreset.PresetID(),
		TrainingConfig:  FinalizedTrainingConfig{Values: values},
	}
}

// CanonicalJSON returns deterministic JSON for comparison and idempotency checks.
func CanonicalJSON(finalized *FinalizedRunSpec) (string, error) {
	data, err := json.Marshal(finalized)
	if err != nil {
		return "", fmt.Errorf("canonicalize finalized spec %s: %w", finalized.SpecID, err)
	}
	return string(data), nil
}

func cloneFinalizedValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, item := range typed {
			cloned[key] = cloneFinalizedValue(item)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for i, item := range typed {
			cloned[i] = cloneFinalizedValue(item)
		}
		return cloned
	default:
		return value
	}
}
