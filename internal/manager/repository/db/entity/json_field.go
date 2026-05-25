package entity

import (
	"fmt"

	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
)

// JSONField scans a database TEXT column that stores a JSON-encoded value into
// a typed Go value. Reserved for cases where a JSON blob column is truly
// unavoidable (see `CLAUDE.md` Schema Rules); new schema should normalize
// instead.
type JSONField[T any] struct {
	Data T
}

// Scan decodes a JSON database field into the typed value.
func (f *JSONField[T]) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return encoding.UnmarshalJSON(v, &f.Data)
	case []byte:
		return encoding.UnmarshalJSON(string(v), &f.Data)
	default:
		return fmt.Errorf("scan json field from %T", src)
	}
}
