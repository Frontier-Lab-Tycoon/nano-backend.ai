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

// TrainingOptions holds preset-validated training overrides.
//
// Keys and value types are validated against the active preset's schema at
// submission time, so the server does not enforce a static struct here.
// Typical entries: learning_rate, num_epochs, lora_r, max_seq_length.
type TrainingOptions struct {
	Overrides map[string]any `json:"overrides,omitempty"`
}
