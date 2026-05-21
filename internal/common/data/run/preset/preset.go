package preset

import (
	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
)

// ID is the stable identity for a preset.
type ID = uuid.UUID

// Category classifies a preset by the part of a Draft or Spec it configures.
type Category string

const (
	// TrainerPreset configures trainer runtime and training parameters.
	TrainerPreset Category = "trainer"
	// ResourcePreset configures resource defaults and policy.
	ResourcePreset Category = "resource"
	// OutputPreset configures output and artifact defaults and policy.
	OutputPreset Category = "output"
)

// Refs selects optional preset categories.
type Refs struct {
	Trainer  *ID `json:"trainer,omitempty"`
	Resource *ID `json:"resource,omitempty"`
	Output   *ID `json:"output,omitempty"`
}

// Options is the resolved option data contributed by a preset.
type Options struct {
	Model              *ModelOptions
	Data               *DataOptions
	Resource           *ResourceOptions
	TrainingParameters map[string]any
}

// ModelOptions describes preset-provided base model options.
type ModelOptions struct {
	BaseModel string `json:"base_model"`
}

// DataOptions describes preset-provided dataset options.
type DataOptions struct {
	Datasets []DatasetRef `json:"datasets"`
}

// DatasetRef identifies a preset-provided dataset and split.
type DatasetRef struct {
	Path  string `json:"path"`
	Split string `json:"split"`
}

// ResourceOptions specifies preset-provided compute resources.
type ResourceOptions struct {
	CPU     run.CPUOptions     `json:"cpu,omitempty"`
	GPU     run.GPUOptions     `json:"gpu"`
	Memory  run.MemoryOptions  `json:"memory"`
	Timeout run.TimeoutOptions `json:"timeout"`
}

// Preset is the resolved preset surface needed by Spec finalization.
type Preset interface {
	PresetID() ID
	Options() Options
}

// Presets contains the resolved presets used to finalize a Spec.
type Presets struct {
	Trainer  Preset
	Resource Preset
	Output   Preset
}

// All returns resolved presets in deterministic category order.
func (p Presets) All() []Preset {
	return []Preset{p.Trainer, p.Resource, p.Output}
}
