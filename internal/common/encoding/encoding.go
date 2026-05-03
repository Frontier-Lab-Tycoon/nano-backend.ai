// Package encoding contains small serialization helpers shared across packages.
package encoding

import (
	"encoding/json"
	"fmt"
	"time"
)

// MarshalJSON serializes a value to a JSON string.
func MarshalJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(data), nil
}

// UnmarshalJSON deserializes a JSON string into v.
func UnmarshalJSON(data string, v any) error {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}
	return nil
}

// FormatTime formats a time in UTC using RFC3339Nano.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// ParseTime parses an RFC3339Nano timestamp.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t, nil
}
