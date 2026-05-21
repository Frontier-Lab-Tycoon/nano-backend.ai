package run

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
