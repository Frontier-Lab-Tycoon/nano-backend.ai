package preset

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

func TestPhase0RegistryGetsPresetsByID(t *testing.T) {
	ctx := context.Background()
	registry := NewPhase0Registry()

	axolotl, err := registry.Get(ctx, PresetAxolotlLoRASFT)
	if err != nil {
		t.Fatalf("get axolotl preset: %v", err)
	}
	if axolotl.PresetID() != PresetAxolotlLoRASFT {
		t.Fatalf("got preset id %s, want %s", axolotl.PresetID(), PresetAxolotlLoRASFT)
	}

	unsloth, err := registry.Get(ctx, PresetUnslothLoRASFT)
	if err != nil {
		t.Fatalf("get unsloth preset: %v", err)
	}
	if unsloth.PresetID() != PresetUnslothLoRASFT {
		t.Fatalf("got preset id %s, want %s", unsloth.PresetID(), PresetUnslothLoRASFT)
	}
}

func TestStaticRegistryReturnsNotFoundForUnknownID(t *testing.T) {
	_, err := NewPhase0Registry().Get(context.Background(), uuid.MustParse("20d05e33-4040-4188-9291-f81d11eb2075"))
	if err == nil {
		t.Fatalf("expected error for unknown preset")
	}
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got error %v, want not_found", err)
	}
}

func TestStaticRegistryListIsOrderedByID(t *testing.T) {
	got, err := NewPhase0Registry().List(context.Background())
	if err != nil {
		t.Fatalf("list presets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d presets, want 2", len(got))
	}
	if got[0].PresetID() != PresetAxolotlLoRASFT || got[1].PresetID() != PresetUnslothLoRASFT {
		t.Fatalf("got preset order %s, %s", got[0].PresetID(), got[1].PresetID())
	}
}

func TestTrainerPresetReturnsCopies(t *testing.T) {
	trainerPreset := AxolotlLoRASFT()

	defaults := trainerPreset.Options().TrainingParameters
	defaults["learning_rate"] = 9.9
	if got := trainerPreset.Options().TrainingParameters["learning_rate"]; got == 9.9 {
		t.Fatalf("mutating returned defaults changed preset defaults")
	}

	policy := trainerPreset.OptionPolicy()
	policy.Rules["learning_rate"] = OptionRule{Type: OptionString}
	if got := trainerPreset.OptionPolicy().Rules["learning_rate"].Type; got != OptionFloat {
		t.Fatalf("got rule type %q, want %q", got, OptionFloat)
	}

	policy = trainerPreset.OptionPolicy()
	*policy.Rules["learning_rate"].Max = 9.9
	if got := *trainerPreset.OptionPolicy().Rules["learning_rate"].Max; got == 9.9 {
		t.Fatalf("mutating returned policy bounds changed preset policy")
	}
}
