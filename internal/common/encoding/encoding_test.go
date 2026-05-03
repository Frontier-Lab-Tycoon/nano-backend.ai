package encoding

import (
	"strings"
	"testing"
	"time"
)

func TestMarshalJSONSuccess(t *testing.T) {
	got, err := MarshalJSON(map[string]int{"b": 2, "a": 1})
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	if got != `{"a":1,"b":2}` {
		t.Fatalf("got %s, want canonical map JSON", got)
	}
}

func TestMarshalJSONError(t *testing.T) {
	_, err := MarshalJSON(func() {})
	if err == nil {
		t.Fatalf("MarshalJSON returned nil error for unsupported value")
	}
	if !strings.Contains(err.Error(), "marshal json") {
		t.Fatalf("got error %q, want marshal json context", err.Error())
	}
}

func TestUnmarshalJSONSuccess(t *testing.T) {
	var got map[string]int
	if err := UnmarshalJSON(`{"a":1}`, &got); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if got["a"] != 1 {
		t.Fatalf("got %v, want map with a=1", got)
	}
}

func TestUnmarshalJSONError(t *testing.T) {
	var got map[string]int
	err := UnmarshalJSON(`{`, &got)
	if err == nil {
		t.Fatalf("UnmarshalJSON returned nil error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal json") {
		t.Fatalf("got error %q, want unmarshal json context", err.Error())
	}
}

func TestFormatTime(t *testing.T) {
	local := time.Date(2026, 5, 3, 10, 30, 0, 123, time.FixedZone("KST", 9*60*60))

	got := FormatTime(local)
	if got != "2026-05-03T01:30:00.000000123Z" {
		t.Fatalf("got %q, want UTC RFC3339Nano timestamp", got)
	}
}

func TestParseTimeSuccess(t *testing.T) {
	got, err := ParseTime("2026-05-03T01:30:00.000000123Z")
	if err != nil {
		t.Fatalf("ParseTime returned error: %v", err)
	}

	want := time.Date(2026, 5, 3, 1, 30, 0, 123, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseTimeError(t *testing.T) {
	_, err := ParseTime("not-a-time")
	if err == nil {
		t.Fatalf("ParseTime returned nil error for invalid timestamp")
	}
	if !strings.Contains(err.Error(), "parse time") {
		t.Fatalf("got error %q, want parse time context", err.Error())
	}
}
