package kernel

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewIDGeneratesUniqueIDs(t *testing.T) {
	a := NewID()
	b := NewID()

	if a.String() == b.String() {
		t.Errorf("expected unique IDs, got %q and %q", a, b)
	}
}

func TestNewIDIsNotZero(t *testing.T) {
	id := NewID()
	if id.IsZero() {
		t.Error("expected non-zero ID from NewID()")
	}
}

func TestIDZeroValue(t *testing.T) {
	var id ID
	if !id.IsZero() {
		t.Error("expected zero-value ID to be zero")
	}
}

func TestParseIDValid(t *testing.T) {
	original := NewID()
	parsed, err := ParseID(original.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.String() != original.String() {
		t.Errorf("got %q, want %q", parsed.String(), original.String())
	}
}

func TestParseIDInvalid(t *testing.T) {
	_, err := ParseID("not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID string")
	}
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestParseIDEmpty(t *testing.T) {
	_, err := ParseID("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

func TestIDEquality(t *testing.T) {
	id := NewID()
	a, _ := ParseID(id.String())
	b, _ := ParseID(id.String())

	if a != b {
		t.Errorf("expected %q == %q", a, b)
	}

	c := NewID()
	if a == c {
		t.Errorf("expected %q != %q", a, c)
	}
}

func TestIDJSONRoundtrip(t *testing.T) {
	original := NewID()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded ID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("got %q, want %q", decoded, original)
	}
}

func TestIDJSONInvalid(t *testing.T) {
	var id ID
	err := json.Unmarshal([]byte(`"not-a-uuid"`), &id)
	if err == nil {
		t.Fatal("expected error for invalid UUID in JSON")
	}
}

func TestSpecSerialize(t *testing.T) {
	spec := Spec{Command: []string{"python", "-c", "print('hello')"}}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded Spec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if len(decoded.Command) != 3 {
		t.Fatalf("got %d commands, want 3", len(decoded.Command))
	}
	if decoded.Command[0] != "python" {
		t.Errorf("got command[0] %q, want %q", decoded.Command[0], "python")
	}
}

func TestSpecDeserialize(t *testing.T) {
	input := `{"command":["echo","hello"]}`

	var spec Spec
	if err := json.Unmarshal([]byte(input), &spec); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if len(spec.Command) != 2 {
		t.Fatalf("got %d commands, want 2", len(spec.Command))
	}
	if spec.Command[1] != "hello" {
		t.Errorf("got command[1] %q, want %q", spec.Command[1], "hello")
	}
}

func TestStatusRunningRoundtrip(t *testing.T) {
	status := Running()

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded Status
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if decoded.Type != StatusRunning {
		t.Errorf("got type %q, want %q", decoded.Type, StatusRunning)
	}
}

func TestStatusExitedRoundtrip(t *testing.T) {
	status := Exited(42)

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded Status
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if decoded.Type != StatusExited {
		t.Errorf("got type %q, want %q", decoded.Type, StatusExited)
	}
	if decoded.Code != 42 {
		t.Errorf("got code %d, want 42", decoded.Code)
	}
}

func TestStatusFailedRoundtrip(t *testing.T) {
	status := Failed("out of memory")

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded Status
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if decoded.Type != StatusFailed {
		t.Errorf("got type %q, want %q", decoded.Type, StatusFailed)
	}
	if decoded.Reason != "out of memory" {
		t.Errorf("got reason %q, want %q", decoded.Reason, "out of memory")
	}
}

func TestErrorNotFound(t *testing.T) {
	id := NewID()
	err := &Error{Op: "status", ID: id, Err: ErrNotFound}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected errors.Is(err, ErrNotFound) to be true")
	}

	want := "kernel status " + id.String() + ": kernel not found"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorAlreadyExists(t *testing.T) {
	id := NewID()
	err := &Error{Op: "create", ID: id, Err: ErrAlreadyExists}

	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("expected errors.Is(err, ErrAlreadyExists) to be true")
	}

	want := "kernel create " + id.String() + ": kernel already exists"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorRuntime(t *testing.T) {
	err := &Error{Op: "create", Err: ErrRuntime}

	if !errors.Is(err, ErrRuntime) {
		t.Errorf("expected errors.Is(err, ErrRuntime) to be true")
	}

	want := "kernel create: kernel runtime error"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorWithZeroID(t *testing.T) {
	var zeroID ID
	err := &Error{Op: "destroy", ID: zeroID, Err: ErrNotFound}

	want := "kernel destroy: kernel not found"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
