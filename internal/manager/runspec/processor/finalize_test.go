package processor

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	runspecpreset "github.com/seedspirit/nano-backend.ai/internal/manager/runspec/preset"
)

func TestFinalizeRunSpecMergesDefaultsAndParameters(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := runspecpreset.AxolotlLoRASFT()

	finalized := FinalizeRunSpec(Candidate{Draft: &runDraft, Presets: preset.Presets{Trainer: &trainerPreset}})

	values := finalized.TrainingOptions.Parameters
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
	runDraft := sampleDraft()
	defaults := map[string]any{
		"lora_r": 16,
		"nested": map[string]any{
			"rank": 8,
		},
	}
	trainerPreset := testPreset{
		id:       runspecpreset.PresetAxolotlLoRASFT,
		defaults: defaults,
	}

	finalized := FinalizeRunSpec(Candidate{Draft: &runDraft, Presets: preset.Presets{Trainer: trainerPreset}})
	finalized.TrainingOptions.Parameters["lora_r"] = 64
	nested, ok := finalized.TrainingOptions.Parameters["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested parameter has type %T, want map[string]any", finalized.TrainingOptions.Parameters["nested"])
	}
	nested["rank"] = 16

	if got := defaults["lora_r"]; got != 16 {
		t.Fatalf("got preset default lora_r %v, want unchanged 16", got)
	}
	nestedDefault, ok := defaults["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested default has type %T, want map[string]any", defaults["nested"])
	}
	if got := nestedDefault["rank"]; got != 8 {
		t.Fatalf("got preset default nested.rank %v, want unchanged 8", got)
	}
}

func TestCanonicalJSONIsDeterministic(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := runspecpreset.AxolotlLoRASFT()

	finalized := FinalizeRunSpec(Candidate{Draft: &runDraft, Presets: preset.Presets{Trainer: &trainerPreset}})
	first, err := CanonicalJSON(&finalized)
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}
	for i := 0; i < 100; i++ {
		finalized := FinalizeRunSpec(Candidate{Draft: &runDraft, Presets: preset.Presets{Trainer: &trainerPreset}})
		got, err := CanonicalJSON(&finalized)
		if err != nil {
			t.Fatalf("canonical json iteration %d: %v", i, err)
		}
		if got != first {
			t.Fatalf("canonical json changed on iteration %d:\nfirst: %s\ngot:   %s", i, first, got)
		}
	}
}

func TestFinalizeRunSpecAppliesPresetDataAndKeepsPresetRefs(t *testing.T) {
	runDraft := sampleDraft()
	trainerPreset := runspecpreset.AxolotlLoRASFT()
	resourcePreset := optionPreset{
		testPreset: testPreset{id: uuid.MustParse("4925e535-5afa-47ca-bd48-dad90d10954c")},
		model:      preset.ModelOptions{BaseModel: "preset/model"},
		data: preset.DataOptions{
			Datasets: []preset.DatasetRef{
				{Path: "preset/data", Split: "validation"},
			},
		},
		resources: preset.ResourceOptions{
			CPU:     run.CPUOptions{Cores: 8},
			GPU:     run.GPUOptions{Count: 4},
			Memory:  run.MemoryOptions{LimitBytes: 68719476736},
			Timeout: run.TimeoutOptions{DurationSeconds: 28800},
		},
	}
	outputPreset := testPreset{id: uuid.MustParse("7f973e96-5bed-47d4-805e-b4d99c8638cf")}

	finalized := FinalizeRunSpec(Candidate{Draft: &runDraft, Presets: preset.Presets{
		Trainer:  &trainerPreset,
		Resource: &resourcePreset,
		Output:   outputPreset,
	}})

	if got := finalized.TrainingOptions.Parameters["lora_alpha"]; got != 32 {
		t.Fatalf("got lora_alpha %v, want trainer preset default 32", got)
	}
	if got := finalized.ModelOptions.BaseModel; got != "preset/model" {
		t.Fatalf("got base model %q, want preset/model", got)
	}
	if got := finalized.DataOptions.Datasets[0].Path; got != "preset/data" {
		t.Fatalf("got dataset path %q, want preset/data", got)
	}
	if got := finalized.ResourceOptions.GPU.Count; got != 4 {
		t.Fatalf("got gpu count %d, want preset override 4", got)
	}
	if finalized.PresetRefs.Trainer == nil || *finalized.PresetRefs.Trainer != runspecpreset.PresetAxolotlLoRASFT {
		t.Fatalf("got trainer preset ref %v, want %s", finalized.PresetRefs.Trainer, runspecpreset.PresetAxolotlLoRASFT)
	}
	data, err := CanonicalJSON(&finalized)
	if err != nil {
		t.Fatalf("canonical json: %v", err)
	}
	if !strings.Contains(data, "preset_refs") {
		t.Fatalf("finalized spec does not contain preset refs: %s", data)
	}
}

func sampleDraft() draft.Draft {
	return draft.Draft{
		ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ProjectID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name:        "mergeowl-exp-42",
		Description: "LoRA SFT experiment",
		PresetRefs: preset.Refs{
			Trainer: runPresetIDPtr(runspecpreset.PresetAxolotlLoRASFT),
		},
		ModelOptions: draft.ModelOptionsReq{
			BaseModel: "unsloth/Llama-3.1-8B",
		},
		DataOptions: draft.DataOptionsReq{
			Datasets: []draft.DatasetRefReq{
				{Path: "mergeowl/v1", Split: "train"},
			},
		},
		ResourceOptions: draft.ResourceOptionsReq{
			GPU:     run.GPUOptions{Count: 1},
			Memory:  run.MemoryOptions{LimitBytes: 34359738368},
			Timeout: run.TimeoutOptions{DurationSeconds: 14400},
		},
		TrainingOptions: draft.TrainingOptionsReq{
			Parameters: map[string]any{
				"learning_rate":  2.0e-4,
				"lora_r":         32,
				"max_seq_length": 4096,
				"num_epochs":     3,
			},
		},
	}
}

func runPresetIDPtr(id preset.ID) *preset.ID {
	return &id
}

type testPreset struct {
	id       preset.ID
	defaults map[string]any
}

func (p testPreset) PresetID() preset.ID {
	return p.id
}

func (p testPreset) Options() preset.Options {
	return preset.Options{TrainingParameters: p.defaults}
}

type optionPreset struct {
	testPreset
	model     preset.ModelOptions
	data      preset.DataOptions
	resources preset.ResourceOptions
}

func (p *optionPreset) Options() preset.Options {
	return preset.Options{
		Model:              &p.model,
		Data:               &p.data,
		Resource:           &p.resources,
		TrainingParameters: p.defaults,
	}
}
