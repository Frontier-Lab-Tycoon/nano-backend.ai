package run

import "github.com/google/uuid"

// ArtifactIndex records the files produced by a Run.
//
// Every Run writes a fixed set of files (spec.yaml, resolved_config.yaml,
// metrics.json, report.md) plus optional outputs (adapter weights, merged
// weights, logs) under a well-known directory structure rooted at BasePath.
//
// The filesystem at BasePath is the source of truth; the Files slice is an
// index used for listing, integrity checks (sha256), and API responses. A
// ArtifactIndex does not need to enumerate files until they are written.
type ArtifactIndex struct {
	RunID    uuid.UUID      `json:"run_id"`
	BasePath string         `json:"base_path"`
	Files    []ArtifactFile `json:"files,omitempty"`
}

// ArtifactFile is a single entry in an ArtifactIndex.
//
// Path is relative to the owning ArtifactIndex's BasePath. SizeBytes and SHA256
// are recorded after the file is materialized so that downloads can be
// verified end-to-end.
type ArtifactFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// NewArtifactIndex creates an ArtifactIndex for the given Run rooted at basePath.
// The Files slice starts empty; entries are appended as files are written.
func NewArtifactIndex(runID uuid.UUID, basePath string) ArtifactIndex {
	return ArtifactIndex{
		RunID:    runID,
		BasePath: basePath,
	}
}
