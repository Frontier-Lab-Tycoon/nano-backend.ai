package entity

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
)

// Spec is the database record shape for a spec row.
type Spec struct {
	ID                             string `db:"id"`
	ProjectID                      string `db:"project_id"`
	Name                           string `db:"name"`
	Description                    string `db:"description"`
	ModelBaseModel                 string `db:"model_base_model"`
	ResourceCPUCores               int    `db:"resource_cpu_cores"`
	ResourceGPUCount               int    `db:"resource_gpu_count"`
	ResourceMemoryLimitBytes       int64  `db:"resource_memory_limit_bytes"`
	ResourceTimeoutDurationSeconds int64  `db:"resource_timeout_duration_seconds"`
	CreatedAt                      string `db:"created_at"`

	PresetRefs         preset.Refs
	Datasets           []SpecDataset
	TrainingParameters []SpecTrainingParameter
}

// SpecDataset is the database record shape for a spec_datasets row.
type SpecDataset struct {
	Ordinal    int    `db:"ordinal"`
	DatasetRef string `db:"dataset_ref"`
	SplitName  string `db:"split_name"`
}

// SpecTrainingParameter is the database record shape for a spec_training_parameters row.
//
// Value is a JSON number literal (e.g. "3", "0.0002"). The server keeps the
// stored representation as a string so it does not commit to int vs float;
// consumers that need a typed view cast via json.Number.
type SpecTrainingParameter struct {
	Key   string `db:"key"`
	Value string `db:"value"`
}

// ToData converts the database record into the public spec type.
func (s *Spec) ToData() (spec.Spec, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return spec.Spec{}, fmt.Errorf("parse spec id %q: %w", s.ID, err)
	}
	projectID, err := uuid.Parse(s.ProjectID)
	if err != nil {
		return spec.Spec{}, fmt.Errorf("parse project id %q: %w", s.ProjectID, err)
	}

	datasets := make([]spec.DatasetRef, 0, len(s.Datasets))
	for _, ds := range s.Datasets {
		datasets = append(datasets, spec.DatasetRef{
			Path:  ds.DatasetRef,
			Split: ds.SplitName,
		})
	}

	parameters := make(map[string]any, len(s.TrainingParameters))
	for _, p := range s.TrainingParameters {
		parameters[p.Key] = json.Number(p.Value)
	}

	return spec.Spec{
		ID:          id,
		ProjectID:   projectID,
		Name:        s.Name,
		Description: s.Description,
		PresetRefs:  s.PresetRefs,
		ModelOptions: spec.ModelOptions{
			BaseModel: s.ModelBaseModel,
		},
		DataOptions: spec.DataOptions{
			Datasets: datasets,
		},
		ResourceOptions: spec.ResourceOptions{
			CPU:     run.CPUOptions{Cores: s.ResourceCPUCores},
			GPU:     run.GPUOptions{Count: s.ResourceGPUCount},
			Memory:  run.MemoryOptions{LimitBytes: s.ResourceMemoryLimitBytes},
			Timeout: run.TimeoutOptions{DurationSeconds: s.ResourceTimeoutDurationSeconds},
		},
		TrainingOptions: spec.TrainingOptions{
			Parameters: parameters,
		},
	}, nil
}

// FromData converts a common Spec into the database entity form.
func FromData(source *spec.Spec, createdAt string) (Spec, error) {
	if source.ID == uuid.Nil {
		return Spec{}, fmt.Errorf("spec id is required")
	}
	if source.ProjectID == uuid.Nil {
		return Spec{}, fmt.Errorf("spec %s: project id is required", source.ID)
	}

	datasets := make([]SpecDataset, 0, len(source.DataOptions.Datasets))
	for i, ds := range source.DataOptions.Datasets {
		datasets = append(datasets, SpecDataset{
			Ordinal:    i,
			DatasetRef: ds.Path,
			SplitName:  ds.Split,
		})
	}

	parameters := make([]SpecTrainingParameter, 0, len(source.TrainingOptions.Parameters))
	for key, value := range source.TrainingOptions.Parameters {
		serialized, err := encoding.FormatNumber(value)
		if err != nil {
			return Spec{}, fmt.Errorf("spec %s training parameter %s: %w", source.ID, key, err)
		}
		parameters = append(parameters, SpecTrainingParameter{
			Key:   key,
			Value: serialized,
		})
	}

	return Spec{
		ID:                             source.ID.String(),
		ProjectID:                      source.ProjectID.String(),
		Name:                           source.Name,
		Description:                    source.Description,
		ModelBaseModel:                 source.ModelOptions.BaseModel,
		ResourceCPUCores:               source.ResourceOptions.CPU.Cores,
		ResourceGPUCount:               source.ResourceOptions.GPU.Count,
		ResourceMemoryLimitBytes:       source.ResourceOptions.Memory.LimitBytes,
		ResourceTimeoutDurationSeconds: source.ResourceOptions.Timeout.DurationSeconds,
		CreatedAt:                      createdAt,
		PresetRefs:                     source.PresetRefs,
		Datasets:                       datasets,
		TrainingParameters:             parameters,
	}, nil
}
