package runspec

import (
	"context"

	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	"github.com/seedspirit/nano-backend.ai/internal/manager/preset"
)

// Processor finalizes a submitted RunSpec for one submission mode.
type Processor interface {
	Process(ctx context.Context, spec *run.Spec) (FinalizedRunSpec, error)
}

// PresetRegistry is the preset lookup dependency consumed by PresetBackedProcessor.
type PresetRegistry interface {
	Get(ctx context.Context, id preset.ID) (preset.Preset, error)
}

// Validator validates a submitted RunSpec against a preset contract.
type Validator interface {
	Validate(spec *run.Spec, trainerPreset preset.Preset) ValidationErrors
}

// PresetBackedProcessor orchestrates preset lookup, validation, and finalization.
type PresetBackedProcessor struct {
	Registry  PresetRegistry
	Validator Validator
}

// Process validates and finalizes a submitted RunSpec.
func (p PresetBackedProcessor) Process(ctx context.Context, spec *run.Spec) (FinalizedRunSpec, error) {
	if p.Registry == nil {
		return FinalizedRunSpec{}, errordef.Errorf(errordef.InvalidInput, "preset registry is nil")
	}
	if p.Validator == nil {
		return FinalizedRunSpec{}, errordef.Errorf(errordef.InvalidInput, "runspec validator is nil")
	}

	presetID, err := ExtractTrainerPresetID(spec)
	if err != nil {
		return FinalizedRunSpec{}, err
	}
	trainerPreset, err := p.Registry.Get(ctx, presetID)
	if err != nil {
		return FinalizedRunSpec{}, err
	}
	if validationErrs := p.Validator.Validate(spec, trainerPreset); validationErrs.HasAny() {
		return FinalizedRunSpec{}, validationErrs
	}
	return FinalizeRunSpec(spec, trainerPreset), nil
}

// ExtractTrainerPresetID reads the trainer preset selector from preset refs.
func ExtractTrainerPresetID(spec *run.Spec) (preset.ID, error) {
	if spec == nil {
		return "", errordef.Errorf(errordef.InvalidInput, "spec is nil")
	}
	if spec.Presets.Trainer == nil || *spec.Presets.Trainer == "" {
		return "", errordef.Errorf(errordef.InvalidInput, "preset_refs.trainer is required")
	}
	return preset.ID(*spec.Presets.Trainer), nil
}
