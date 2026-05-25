package specbuilder

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	runspecpreset "github.com/seedspirit/nano-backend.ai/internal/manager/runspec/preset"
)

func TestPresetBackedOrchestratesLookupValidationAndFinalize(t *testing.T) {
	ctx := context.Background()
	runDraft := sampleDraft()
	trainerPreset := runspecpreset.AxolotlLoRASFT()
	resourcePreset := testPreset{id: uuid.MustParse("4925e535-5afa-47ca-bd48-dad90d10954c")}
	runDraft.PresetRefs.Resource = runPresetIDPtr(resourcePreset.PresetID())
	registry := &recordingRegistry{
		presets: map[preset.ID]preset.Preset{
			runspecpreset.PresetAxolotlLoRASFT: &trainerPreset,
			resourcePreset.PresetID():          resourcePreset,
		},
	}
	validator := &recordingValidator{}
	builder := PresetBacked{
		Registry:  registry,
		Validator: validator,
	}

	finalized, err := builder.Build(ctx, &runDraft)
	if err != nil {
		t.Fatalf("build runspec: %v", err)
	}

	if registry.callCount != 1 {
		t.Fatalf("got registry call count %d, want 1", registry.callCount)
	}
	if len(registry.requestedIDs) != 2 {
		t.Fatalf("got registry ids %v, want trainer and resource ids in one lookup", registry.requestedIDs)
	}
	if registry.requestedIDs[0] != runspecpreset.PresetAxolotlLoRASFT {
		t.Fatalf("got first registry id %s, want %s", registry.requestedIDs[0], runspecpreset.PresetAxolotlLoRASFT)
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
	if validator.presetID != runspecpreset.PresetAxolotlLoRASFT {
		t.Fatalf("validator got preset id %s, want %s", validator.presetID, runspecpreset.PresetAxolotlLoRASFT)
	}
	if validator.resourcePresetID != resourcePreset.PresetID() {
		t.Fatalf("validator got resource preset id %s, want %s", validator.resourcePresetID, resourcePreset.PresetID())
	}
	if finalized.ID != runDraft.ID {
		t.Fatalf("got finalized spec id %s, want %s", finalized.ID, runDraft.ID)
	}
}

func TestPresetBackedStopsOnValidatorError(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := runspecpreset.AxolotlLoRASFT()
	wantErr := errordef.Errorf(errordef.ValidationError, "training_options.parameters.lora_r: out of range")
	builder := PresetBacked{
		Registry: &recordingRegistry{
			presets: map[preset.ID]preset.Preset{
				runspecpreset.PresetAxolotlLoRASFT: &trainerPreset,
			},
		},
		Validator: &recordingValidator{err: wantErr},
	}

	finalized, err := builder.Build(context.Background(), &runDraft)
	if err == nil {
		t.Fatalf("expected validator error")
	}
	if !errors.Is(err, errordef.ErrValidation) {
		t.Fatalf("got error %v, want ErrValidation", err)
	}
	if finalized.ID != uuid.Nil {
		t.Fatalf("got finalized output %+v, want zero value", finalized)
	}
}

func TestPresetBackedReturnsRegistryError(t *testing.T) {
	runDraft := sampleDraft()
	builder := PresetBacked{
		Registry:  &recordingRegistry{err: errordef.ErrNotFound},
		Validator: &recordingValidator{},
	}

	_, err := builder.Build(context.Background(), &runDraft)
	if err == nil {
		t.Fatalf("expected registry error")
	}
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got error %v, want not_found", err)
	}
}

func TestReadPresets(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := runspecpreset.AxolotlLoRASFT()
	builder := PresetBacked{
		Registry: &recordingRegistry{
			presets: map[preset.ID]preset.Preset{
				runspecpreset.PresetAxolotlLoRASFT: &trainerPreset,
			},
		},
	}

	got, err := builder.readPresets(context.Background(), &runDraft)
	if err != nil {
		t.Fatalf("read presets: %v", err)
	}
	if got.Trainer == nil || got.Trainer.PresetID() != runspecpreset.PresetAxolotlLoRASFT {
		t.Fatalf("got trainer preset %v, want %s", got.Trainer, runspecpreset.PresetAxolotlLoRASFT)
	}
}

func TestReadPresetsAllowsMissingTrainerSelector(t *testing.T) {
	runDraft := sampleDraft()
	runDraft.PresetRefs.Trainer = nil
	builder := PresetBacked{}

	got, err := builder.readPresets(context.Background(), &runDraft)
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
	err              error
}

func (v *recordingValidator) Validate(candidate Candidate) error {
	v.called = true
	v.draft = candidate.Draft
	if candidate.Presets.Trainer != nil {
		v.presetID = candidate.Presets.Trainer.PresetID()
	}
	if candidate.Presets.Resource != nil {
		v.resourcePresetID = candidate.Presets.Resource.PresetID()
	}
	return v.err
}
