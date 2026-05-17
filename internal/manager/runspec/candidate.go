package runspec

import (
	"github.com/seedspirit/nano-backend.ai/internal/common/run/draft"
	"github.com/seedspirit/nano-backend.ai/internal/common/run/preset"
)

// Candidate is the validation target created from user draft and resolved presets.
type Candidate struct {
	Draft   *draft.Draft
	Presets preset.Presets
}
