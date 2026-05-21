package entity

import (
	"fmt"

	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
)

type jsonField[T any] struct {
	Data T
}

// Scan decodes a JSON database field into the typed value.
func (f *jsonField[T]) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return encoding.UnmarshalJSON(v, &f.Data)
	case []byte:
		return encoding.UnmarshalJSON(string(v), &f.Data)
	default:
		return fmt.Errorf("scan json field from %T", src)
	}
}
