package specbuilder

import (
	"encoding/json"
	"fmt"

	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
)

// FinalizeRunSpec combines a validated candidate into an immutable run Spec.
func FinalizeRunSpec(candidate Candidate) spec.Spec {
	return overridePresets(candidate)
}

func overridePresets(candidate Candidate) spec.Spec {
	runDraft := candidate.Draft
	runSpec := spec.Spec{
		ID:              runDraft.ID,
		ProjectID:       runDraft.ProjectID,
		Name:            runDraft.Name,
		Description:     runDraft.Description,
		PresetRefs:      runDraft.PresetRefs,
		ModelOptions:    modelOptionsFromReq(runDraft.ModelOptions),
		DataOptions:     dataOptionsFromReq(runDraft.DataOptions),
		ResourceOptions: resourceOptionsFromReq(runDraft.ResourceOptions),
		TrainingOptions: spec.TrainingOptions{Parameters: map[string]any{}},
	}

	overridePresetOptions(&runSpec, candidate.Presets)
	overrideDraftOptions(&runSpec, runDraft)

	return runSpec
}

func overridePresetOptions(runSpec *spec.Spec, presets preset.Presets) {
	for _, resolvedPreset := range presets.All() {
		if resolvedPreset == nil {
			continue
		}
		options := resolvedPreset.Options()
		if options.Model != nil {
			runSpec.ModelOptions = modelOptionsFromPreset(*options.Model)
		}
		if options.Data != nil {
			runSpec.DataOptions = dataOptionsFromPreset(*options.Data)
		}
		if options.Resource != nil {
			runSpec.ResourceOptions = resourceOptionsFromPreset(*options.Resource)
		}
		for key, value := range options.TrainingParameters {
			runSpec.TrainingOptions.Parameters[key] = copyValue(value)
		}
	}
}

func overrideDraftOptions(runSpec *spec.Spec, runDraft *draft.Draft) {
	for key, value := range runDraft.TrainingOptions.Parameters {
		runSpec.TrainingOptions.Parameters[key] = copyValue(value)
	}
}

// CanonicalJSON returns deterministic JSON for comparison and idempotency checks.
func CanonicalJSON(runSpec *spec.Spec) (string, error) {
	data, err := json.Marshal(runSpec)
	if err != nil {
		return "", fmt.Errorf("canonicalize spec %s: %w", runSpec.ID, err)
	}
	return string(data), nil
}

func copyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, item := range typed {
			cloned[key] = copyValue(item)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for i, item := range typed {
			cloned[i] = copyValue(item)
		}
		return cloned
	default:
		return value
	}
}

func modelOptionsFromReq(req draft.ModelOptionsReq) spec.ModelOptions {
	return spec.ModelOptions{BaseModel: req.BaseModel}
}

func dataOptionsFromReq(req draft.DataOptionsReq) spec.DataOptions {
	datasets := make([]spec.DatasetRef, len(req.Datasets))
	for i, item := range req.Datasets {
		datasets[i] = spec.DatasetRef{Path: item.Path, Split: item.Split}
	}
	return spec.DataOptions{Datasets: datasets}
}

func resourceOptionsFromReq(req draft.ResourceOptionsReq) spec.ResourceOptions {
	return spec.ResourceOptions{
		CPU:     req.CPU,
		GPU:     req.GPU,
		Memory:  req.Memory,
		Timeout: req.Timeout,
	}
}

func modelOptionsFromPreset(options preset.ModelOptions) spec.ModelOptions {
	return spec.ModelOptions{BaseModel: options.BaseModel}
}

func dataOptionsFromPreset(options preset.DataOptions) spec.DataOptions {
	datasets := make([]spec.DatasetRef, len(options.Datasets))
	for i, item := range options.Datasets {
		datasets[i] = spec.DatasetRef{Path: item.Path, Split: item.Split}
	}
	return spec.DataOptions{Datasets: datasets}
}

func resourceOptionsFromPreset(options preset.ResourceOptions) spec.ResourceOptions {
	return spec.ResourceOptions{
		CPU:     options.CPU,
		GPU:     options.GPU,
		Memory:  options.Memory,
		Timeout: options.Timeout,
	}
}
