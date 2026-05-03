package run

// ResourceOptions specifies the compute resources requested for a Run.
//
// These are *requested* resources captured on the Spec. Allocated values
// (concrete GPU index, host paths, etc.) live on the executor's
// ExecutionPlan and are not part of the Run domain model.
type ResourceOptions struct {
	CPU     CPUOptions     `json:"cpu,omitempty"`
	GPU     GPUOptions     `json:"gpu"`
	Memory  MemoryOptions  `json:"memory"`
	Timeout TimeoutOptions `json:"timeout"`
}

// CPUOptions specifies optional CPU resource requirements.
type CPUOptions struct {
	Cores int `json:"cores,omitempty"`
}

// GPUOptions specifies GPU resource requirements.
//
// Count is the logical number of GPUs requested, not a device index.
type GPUOptions struct {
	Count int `json:"count"`
}

// MemoryOptions specifies memory resource requirements in bytes.
type MemoryOptions struct {
	LimitBytes int64 `json:"limit_bytes"`
}

// TimeoutOptions specifies the maximum run duration in seconds.
type TimeoutOptions struct {
	DurationSeconds int64 `json:"duration_seconds"`
}
