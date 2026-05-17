package runspec

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	trainerpreset "github.com/seedspirit/nano-backend.ai/internal/manager/preset"
)

func TestPresetBackedProcessorOrchestratesLookupValidationAndFinalize(t *testing.T) {
	ctx := context.Background()
	runDraft := sampleDraft()
	trainerPreset := trainerpreset.AxolotlLoRASFT()
	resourcePreset := testPreset{id: uuid.MustParse("4925e535-5afa-47ca-bd48-dad90d10954c")}
	runDraft.PresetRefs.Resource = runPresetIDPtr(resourcePreset.PresetID())
	registry := &recordingRegistry{
		presets: map[preset.ID]preset.Preset{
			trainerpreset.PresetAxolotlLoRASFT: &trainerPreset,
			resourcePreset.PresetID():          resourcePreset,
		},
	}
	validator := &recordingValidator{}
	processor := PresetBackedProcessor{
		Registry:  registry,
		Validator: validator,
	}

	finalized, err := processor.Process(ctx, &runDraft)
	if err != nil {
		t.Fatalf("process runspec: %v", err)
	}

	if registry.callCount != 1 {
		t.Fatalf("got registry call count %d, want 1", registry.callCount)
	}
	if len(registry.requestedIDs) != 2 {
		t.Fatalf("got registry ids %v, want trainer and resource ids in one lookup", registry.requestedIDs)
	}
	if registry.requestedIDs[0] != trainerpreset.PresetAxolotlLoRASFT {
		t.Fatalf("got first registry id %s, want %s", registry.requestedIDs[0], trainerpreset.PresetAxolotlLoRASFT)
	}
	if registry.requestedIDs[1] != resourcePreset.PresetID() {
		t.Fatalf("got second registry id %s, want %s", registry.requestedIDs[1], resourcePreset.PresetID())
	}
	if !validator.called {
		t.Fatalf("validator was not called")
	}
	if validator.draft.ID != runDraft.ID {
		t.Fatalf("validator got spec id %s, want %s", validator.draft.ID, runDraft.ID)
	}
	if validator.presetID != trainerpreset.PresetAxolotlLoRASFT {
		t.Fatalf("validator got preset id %s, want %s", validator.presetID, trainerpreset.PresetAxolotlLoRASFT)
	}
	if validator.resourcePresetID != resourcePreset.PresetID() {
		t.Fatalf("validator got resource preset id %s, want %s", validator.resourcePresetID, resourcePreset.PresetID())
	}
	if finalized.ID != runDraft.ID {
		t.Fatalf("got finalized spec id %s, want %s", finalized.ID, runDraft.ID)
	}
}

func TestPresetBackedProcessorStopsOnValidationErrors(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := trainerpreset.AxolotlLoRASFT()
	processor := PresetBackedProcessor{
		Registry: &recordingRegistry{
			presets: map[preset.ID]preset.Preset{
				trainerpreset.PresetAxolotlLoRASFT: &trainerPreset,
			},
		},
		Validator: &recordingValidator{
			errs: ValidationErrors{
				{Field: "training_options.parameters.lora_r", Reason: "out of range"},
			},
		},
	}

	finalized, err := processor.Process(context.Background(), &runDraft)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("got error %T %v, want ValidationErrors", err, err)
	}
	if finalized.ID != uuid.Nil {
		t.Fatalf("got finalized output %+v, want zero value", finalized)
	}
}

func TestPresetBackedProcessorReturnsRegistryError(t *testing.T) {
	runDraft := sampleDraft()
	processor := PresetBackedProcessor{
		Registry:  &recordingRegistry{err: errordef.ErrNotFound},
		Validator: &recordingValidator{},
	}

	_, err := processor.Process(context.Background(), &runDraft)
	if err == nil {
		t.Fatalf("expected registry error")
	}
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got error %v, want not_found", err)
	}
}

func TestReadPresets(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := trainerpreset.AxolotlLoRASFT()
	processor := PresetBackedProcessor{
		Registry: &recordingRegistry{
			presets: map[preset.ID]preset.Preset{
				trainerpreset.PresetAxolotlLoRASFT: &trainerPreset,
			},
		},
	}

	got, err := processor.readPresets(context.Background(), &runDraft)
	if err != nil {
		t.Fatalf("read presets: %v", err)
	}
	if got.Trainer == nil || got.Trainer.PresetID() != trainerpreset.PresetAxolotlLoRASFT {
		t.Fatalf("got trainer preset %v, want %s", got.Trainer, trainerpreset.PresetAxolotlLoRASFT)
	}
}

func TestReadPresetsAllowsMissingTrainerSelector(t *testing.T) {
	runDraft := sampleDraft()
	runDraft.PresetRefs.Trainer = nil
	processor := PresetBackedProcessor{}

	got, err := processor.readPresets(context.Background(), &runDraft)
	if err != nil {
		t.Fatalf("read presets: %v", err)
	}
	if got.Trainer != nil {
		t.Fatalf("got trainer preset %v, want nil", got.Trainer)
	}
}

type recordingRegistry struct {
	presets      map[preset.ID]preset.Preset
	requestedIDs []preset.ID
	callCount    int
	err          error
}

func (r *recordingRegistry) GetMany(_ context.Context, ids []preset.ID) (map[preset.ID]preset.Preset, error) {
	r.callCount++
	r.requestedIDs = append(r.requestedIDs, ids...)
	if r.err != nil {
		return nil, r.err
	}
	resolved := make(map[preset.ID]preset.Preset, len(ids))
	for _, id := range ids {
		resolved[id] = r.presets[id]
	}
	return resolved, nil
}

type recordingValidator struct {
	called           bool
	draft            *draft.Draft
	presetID         preset.ID
	resourcePresetID preset.ID
	errs             ValidationErrors
}

func (v *recordingValidator) Validate(candidate Candidate) ValidationErrors {
	v.called = true
	v.draft = candidate.Draft
	if candidate.Presets.Trainer != nil {
		v.presetID = candidate.Presets.Trainer.PresetID()
	}
	if candidate.Presets.Resource != nil {
		v.resourcePresetID = candidate.Presets.Resource.PresetID()
	}
	return v.errs
}
