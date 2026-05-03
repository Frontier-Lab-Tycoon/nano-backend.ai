package record

import "github.com/seedspirit/nano-backend.ai/internal/common/run"

// Artifact is the database record shape for an artifact file row.
type Artifact struct {
	Path      string `db:"path"`
	SizeBytes int64  `db:"size_bytes"`
	SHA256    string `db:"sha256"`
}

// ToArtifactFile converts the database record into the public artifact file type.
func (a Artifact) ToArtifactFile() run.ArtifactFile {
	return run.ArtifactFile{
		Path:      a.Path,
		SizeBytes: a.SizeBytes,
		SHA256:    a.SHA256,
	}
}
