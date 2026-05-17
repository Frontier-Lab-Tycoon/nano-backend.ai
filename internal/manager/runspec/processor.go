package runspec

import (
	"context"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

// Processor finalizes a submitted RunDraft for one submission mode.
type Processor interface {
	Process(ctx context.Context, runDraft *draft.Draft) (spec.Spec, error)
}

// PresetRegistry is the preset lookup dependency consumed by PresetBackedProcessor.
type PresetRegistry interface {
	GetMany(ctx context.Context, ids []preset.ID) (map[preset.ID]preset.Preset, error)
}

// Validator validates a draft plus resolved preset contracts.
type Validator interface {
	Validate(candidate Candidate) ValidationErrors
}

// PresetBackedProcessor orchestrates preset lookup, validation, and finalization.
type PresetBackedProcessor struct {
	Registry  PresetRegistry
	Validator Validator
}

// Process validates and finalizes a submitted RunDraft.
func (p PresetBackedProcessor) Process(ctx context.Context, runDraft *draft.Draft) (spec.Spec, error) {
	if p.Registry == nil {
		return spec.Spec{}, errordef.Errorf(errordef.InvalidInput, "preset registry is nil")
	}
	if p.Validator == nil {
		return spec.Spec{}, errordef.Errorf(errordef.InvalidInput, "runspec validator is nil")
	}

	presets, err := p.readPresets(ctx, runDraft)
	if err != nil {
		return spec.Spec{}, err
	}
	candidate := Candidate{Draft: runDraft, Presets: presets}
	if validationErrs := p.Validator.Validate(candidate); validationErrs.HasAny() {
		return spec.Spec{}, validationErrs
	}
	return FinalizeRunSpec(candidate), nil
}

func (p PresetBackedProcessor) readPresets(ctx context.Context, runDraft *draft.Draft) (preset.Presets, error) {
	if runDraft == nil {
		return preset.Presets{}, errordef.Errorf(errordef.InvalidInput, "draft is nil")
	}
	resolved, err := p.readPreset(ctx, runDraft.PresetRefs)
	if err != nil {
		return preset.Presets{}, err
	}
	return preset.Presets{
		Trainer:  presetByRef(resolved, runDraft.PresetRefs.Trainer),
		Resource: presetByRef(resolved, runDraft.PresetRefs.Resource),
		Output:   presetByRef(resolved, runDraft.PresetRefs.Output),
	}, nil
}

// TODO: Read preset data from database and cache it in memory for efficient lookup
func (p PresetBackedProcessor) readPreset(ctx context.Context, refs preset.Refs) (map[preset.ID]preset.Preset, error) {
	ids := collectPresetIDs(refs)
	if len(ids) == 0 {
		return map[preset.ID]preset.Preset{}, nil
	}
	return p.Registry.GetMany(ctx, ids)
}

func collectPresetIDs(refs preset.Refs) []preset.ID {
	candidates := []*preset.ID{refs.Trainer, refs.Resource, refs.Output}
	ids := make([]preset.ID, 0, len(candidates))
	seen := make(map[preset.ID]struct{}, len(candidates))
	for _, ref := range candidates {
		if ref == nil || *ref == uuid.Nil {
			continue
		}
		if _, ok := seen[*ref]; ok {
			continue
		}
		seen[*ref] = struct{}{}
		ids = append(ids, *ref)
	}
	return ids
}

func presetByRef(resolved map[preset.ID]preset.Preset, ref *preset.ID) preset.Preset {
	if ref == nil || *ref == uuid.Nil {
		return nil
	}
	return resolved[*ref]
}
