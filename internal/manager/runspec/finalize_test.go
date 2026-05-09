package runspec

import (
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/preset"
)

func TestFinalizeRunSpecMergesDefaultsAndParameters(t *testing.T) {
	spec := sampleSpec()
	trainerPreset := preset.AxolotlLoRASFT()

	finalized := FinalizeRunSpec(&spec, &trainerPreset)

	if finalized.TrainerPresetID != preset.PresetAxolotlLoRASFT {
		t.Fatalf("got trainer preset id %q, want %q", finalized.TrainerPresetID, preset.PresetAxolotlLoRASFT)
	}
	values := finalized.TrainingConfig.Values
	if got := values["learning_rate"]; got != 2.0e-4 {
		t.Fatalf("got learning_rate %v, want parameter 2.0e-4", got)
	}
	if got := values["lora_r"]; got != 32 {
		t.Fatalf("got lora_r %v, want parameter 32", got)
	}
	if got := values["lora_alpha"]; got != 32 {
		t.Fatalf("got lora_alpha %v, want default 32", got)
	}
}

func TestFinalizeRunSpecDoesNotMutatePresetDefaults(t *testing.T) {
	spec := sampleSpec()
	trainerPreset := preset.AxolotlLoRASFT()

	_ = FinalizeRunSpec(&spec, &trainerPreset)

	if got := trainerPreset.Defaults()["lora_r"]; got != 16 {
		t.Fatalf("got preset default lora_r %v, want unchanged 16", got)
	}
}

func TestCanonicalJSONIsDeterministic(t *testing.T) {
	spec := sampleSpec()
	trainerPreset := preset.AxolotlLoRASFT()

	finalized := FinalizeRunSpec(&spec, &trainerPreset)
	first, err := CanonicalJSON(&finalized)
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}
	for i := 0; i < 100; i++ {
		finalized := FinalizeRunSpec(&spec, &trainerPreset)
		got, err := CanonicalJSON(&finalized)
		if err != nil {
			t.Fatalf("canonical json iteration %d: %v", i, err)
		}
		if got != first {
			t.Fatalf("canonical json changed on iteration %d:\nfirst: %s\ngot:   %s", i, first, got)
		}
	}
}

func sampleSpec() run.Spec {
	return run.Spec{
		ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ProjectID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name:        "mergeowl-exp-42",
		Description: "LoRA SFT experiment",
		Presets: run.PresetRefs{
			Trainer: runPresetIDPtr(preset.PresetAxolotlLoRASFT),
		},
		ModelOptions: run.ModelOptions{
			BaseModel: "unsloth/Llama-3.1-8B",
		},
		DataOptions: run.DataOptions{
			Datasets: []run.DatasetRef{
				{Path: "mergeowl/v1", Split: "train"},
			},
		},
		ResourceOptions: run.ResourceOptions{
			GPU:     run.GPUOptions{Count: 1},
			Memory:  run.MemoryOptions{LimitBytes: 34359738368},
			Timeout: run.TimeoutOptions{DurationSeconds: 14400},
		},
		TrainingOptions: run.TrainingOptions{
			Parameters: map[string]any{
				"learning_rate":  2.0e-4,
				"lora_r":         32,
				"max_seq_length": 4096,
				"num_epochs":     3,
			},
		},
	}
}

func runPresetIDPtr(id run.PresetID) *run.PresetID {
	return &id
}
