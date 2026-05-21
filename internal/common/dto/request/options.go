package request

import (
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/draft"
)

// ModelOptionsReq describes the requested base model for a run draft.
type ModelOptionsReq struct {
	BaseModel string `json:"base_model"`
}

// DataOptionsReq describes the requested dataset(s) for a run draft.
type DataOptionsReq struct {
	Datasets []DatasetRefReq `json:"datasets"`
}

// DatasetRefReq identifies a requested dataset and split.
type DatasetRefReq struct {
	Path  string `json:"path"`
	Split string `json:"split"`
}

// ResourceOptionsReq specifies requested compute resources for a run draft.
type ResourceOptionsReq struct {
	CPU     run.CPUOptions     `json:"cpu,omitempty"`
	GPU     run.GPUOptions     `json:"gpu"`
	Memory  run.MemoryOptions  `json:"memory"`
	Timeout run.TimeoutOptions `json:"timeout"`
}

// TrainingOptionsReq holds user-provided training parameters.
type TrainingOptionsReq struct {
	Parameters map[string]any `json:"parameters,omitempty"`
}

func modelOptionsToDraft(req ModelOptionsReq) draft.ModelOptionsReq {
	return draft.ModelOptionsReq{BaseModel: req.BaseModel}
}

func dataOptionsToDraft(req DataOptionsReq) draft.DataOptionsReq {
	datasets := make([]draft.DatasetRefReq, 0, len(req.Datasets))
	for _, d := range req.Datasets {
		datasets = append(datasets, draft.DatasetRefReq{Path: d.Path, Split: d.Split})
	}
	return draft.DataOptionsReq{Datasets: datasets}
}

func resourceOptionsToDraft(req ResourceOptionsReq) draft.ResourceOptionsReq {
	return draft.ResourceOptionsReq{
		CPU:     req.CPU,
		GPU:     req.GPU,
		Memory:  req.Memory,
		Timeout: req.Timeout,
	}
}

func trainingOptionsToDraft(req TrainingOptionsReq) draft.TrainingOptionsReq {
	return draft.TrainingOptionsReq{Parameters: req.Parameters}
}
