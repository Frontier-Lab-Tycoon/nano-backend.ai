package processor

import "strings"

// ValidationError describes one candidate validation failure.
type ValidationError struct {
	Field  string
	Reason string
}

// ValidationErrors is an aggregated validation error list.
type ValidationErrors []ValidationError

// HasAny reports whether the list contains at least one validation error.
func (e ValidationErrors) HasAny() bool {
	return len(e) > 0
}

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(e))
	for _, item := range e {
		if item.Field == "" {
			parts = append(parts, item.Reason)
			continue
		}
		parts = append(parts, item.Field+": "+item.Reason)
	}
	return "validation failed: " + strings.Join(parts, "; ")
}
