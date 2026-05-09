package runspec

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	"github.com/seedspirit/nano-backend.ai/internal/manager/preset"
)

func TestPresetBackedProcessorOrchestratesLookupValidationAndFinalize(t *testing.T) {
	ctx := context.Background()
	spec := sampleSpec()
	trainerPreset := preset.AxolotlLoRASFT()
	registry := &recordingRegistry{
		presets: map[preset.ID]preset.Preset{
			preset.PresetAxolotlLoRASFT: &trainerPreset,
		},
	}
	validator := &recordingValidator{}
	processor := PresetBackedProcessor{
		Registry:  registry,
		Validator: validator,
	}

	finalized, err := processor.Process(ctx, &spec)
	if err != nil {
		t.Fatalf("process runspec: %v", err)
	}

	if registry.requestedID != preset.PresetAxolotlLoRASFT {
		t.Fatalf("got registry id %q, want %q", registry.requestedID, preset.PresetAxolotlLoRASFT)
	}
	if !validator.called {
		t.Fatalf("validator was not called")
	}
	if validator.spec.ID != spec.ID {
		t.Fatalf("validator got spec id %s, want %s", validator.spec.ID, spec.ID)
	}
	if validator.presetID != preset.PresetAxolotlLoRASFT {
		t.Fatalf("validator got preset id %q, want %q", validator.presetID, preset.PresetAxolotlLoRASFT)
	}
	if finalized.SpecID != spec.ID {
		t.Fatalf("got finalized spec id %s, want %s", finalized.SpecID, spec.ID)
	}
}

func TestPresetBackedProcessorStopsOnValidationErrors(t *testing.T) {
	spec := sampleSpec()
	trainerPreset := preset.AxolotlLoRASFT()
	processor := PresetBackedProcessor{
		Registry: &recordingRegistry{
			presets: map[preset.ID]preset.Preset{
				preset.PresetAxolotlLoRASFT: &trainerPreset,
			},
		},
		Validator: &recordingValidator{
			errs: ValidationErrors{
				{Field: "training_options.parameters.lora_r", Reason: "out of range"},
			},
		},
	}

	finalized, err := processor.Process(context.Background(), &spec)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("got error %T %v, want ValidationErrors", err, err)
	}
	if finalized.SpecID != uuid.Nil {
		t.Fatalf("got finalized output %+v, want zero value", finalized)
	}
}

func TestPresetBackedProcessorReturnsRegistryError(t *testing.T) {
	spec := sampleSpec()
	processor := PresetBackedProcessor{
		Registry:  &recordingRegistry{err: errordef.ErrNotFound},
		Validator: &recordingValidator{},
	}

	_, err := processor.Process(context.Background(), &spec)
	if err == nil {
		t.Fatalf("expected registry error")
	}
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got error %v, want not_found", err)
	}
}

func TestExtractTrainerPresetID(t *testing.T) {
	spec := sampleSpec()

	got, err := ExtractTrainerPresetID(&spec)
	if err != nil {
		t.Fatalf("extract preset id: %v", err)
	}
	if got != preset.PresetAxolotlLoRASFT {
		t.Fatalf("got preset id %q, want %q", got, preset.PresetAxolotlLoRASFT)
	}
}

func TestExtractTrainerPresetIDRejectsMissingSelector(t *testing.T) {
	spec := sampleSpec()
	spec.Presets.Trainer = nil

	_, err := ExtractTrainerPresetID(&spec)
	if err == nil {
		t.Fatalf("expected missing preset selector error")
	}
	if !errors.Is(err, errordef.ErrInvalidInput) {
		t.Fatalf("got error %v, want invalid_input", err)
	}
}

type recordingRegistry struct {
	presets     map[preset.ID]preset.Preset
	requestedID preset.ID
	err         error
}

func (r *recordingRegistry) Get(_ context.Context, id preset.ID) (preset.Preset, error) {
	r.requestedID = id
	if r.err != nil {
		return nil, r.err
	}
	return r.presets[id], nil
}

type recordingValidator struct {
	called   bool
	spec     *run.Spec
	presetID preset.ID
	errs     ValidationErrors
}

func (v *recordingValidator) Validate(spec *run.Spec, trainerPreset preset.Preset) ValidationErrors {
	v.called = true
	v.spec = spec
	v.presetID = trainerPreset.PresetID()
	return v.errs
}
