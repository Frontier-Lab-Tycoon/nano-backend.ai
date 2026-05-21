package spec

import "github.com/seedspirit/nano-backend.ai/internal/common/data/run"

// ModelOptions describes the finalized base model for a Run.
type ModelOptions struct {
	BaseModel string `json:"base_model"`
}

// DataOptions describes the finalized dataset(s) used by a Run.
type DataOptions struct {
	Datasets []DatasetRef `json:"datasets"`
}

// DatasetRef identifies a finalized dataset and split.
type DatasetRef struct {
	Path  string `json:"path"`
	Split string `json:"split"`
}

// ResourceOptions specifies finalized compute resources for a Run.
type ResourceOptions struct {
	CPU     run.CPUOptions     `json:"cpu,omitempty"`
	GPU     run.GPUOptions     `json:"gpu"`
	Memory  run.MemoryOptions  `json:"memory"`
	Timeout run.TimeoutOptions `json:"timeout"`
}

// TrainingOptions holds finalized training parameters.
type TrainingOptions struct {
	Parameters map[string]any `json:"parameters,omitempty"`
}
