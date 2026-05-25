package validator

import (
	"github.com/seedspirit/nano-backend.ai/internal/manager/runspec/specbuilder"
)

// Noop accepts every candidate.
//
// TODO(#24): replace with the default rule-based validator once preset rule
// enforcement lands.
type Noop struct{}

// Validate returns nil for any candidate.
func (Noop) Validate(specbuilder.Candidate) error {
	return nil
}
