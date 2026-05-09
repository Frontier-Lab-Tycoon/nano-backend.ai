package run

// ModelOptions describes the base model for a Run.
//
// BaseModel is a model reference string — typically a HuggingFace Hub ID
// (e.g., "unsloth/Llama-3.1-8B") or a scheme-prefixed URI (hf://..., local://...).
type ModelOptions struct {
	BaseModel string `json:"base_model"`
}

// DataOptions describes the dataset(s) used by a Run.
type DataOptions struct {
	Datasets []DatasetRef `json:"datasets"`
}

// DatasetRef identifies a dataset and the split to consume.
//
// Path follows the same reference scheme as ModelOptions.BaseModel: a HF Hub
// dataset ID (e.g., "mergeowl/v1") or a scheme-prefixed URI. Split selects
// the partition (e.g., "train", "validation").
type DatasetRef struct {
	Path  string `json:"path"`
	Split string `json:"split"`
}

// PresetID is the stable identity of a preset.
type PresetID string

// PresetCategory classifies a preset by the part of a RunSpec it configures.
type PresetCategory string

const (
	// TrainerPreset configures trainer runtime and training parameters.
	TrainerPreset PresetCategory = "trainer"
	// ResourcePreset configures resource defaults and policy.
	ResourcePreset PresetCategory = "resource"
	// OutputPreset configures output and artifact defaults and policy.
	OutputPreset PresetCategory = "output"
)

// PresetRefs selects optional preset categories for a RunSpec.
type PresetRefs struct {
	Trainer  *PresetID `json:"trainer,omitempty"`
	Resource *PresetID `json:"resource,omitempty"`
	Output   *PresetID `json:"output,omitempty"`
}

// TrainingOptions holds user-provided training parameters.
//
// Parameter keys and value types are interpreted by the selected processor.
// Preset-backed processing validates them against the trainer preset policy.
// Typical keys: learning_rate, num_epochs, lora_r, max_seq_length.
type TrainingOptions struct {
	Parameters map[string]any `json:"parameters,omitempty"`
}
