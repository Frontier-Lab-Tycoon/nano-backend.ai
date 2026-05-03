package errordef

import (
	"errors"
	"net/http"
	"testing"
)

func TestErrorExposesCodeStatusAndMessage(t *testing.T) {
	err := Error(InvalidInput, "missing db path")

	coded, ok := err.(interface {
		Code() string
		StatusCode() int
	})
	if !ok {
		t.Fatalf("error does not expose Code and StatusCode")
	}
	if coded.Code() != "invalid_input" {
		t.Fatalf("got code %q, want invalid_input", coded.Code())
	}
	if coded.StatusCode() != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", coded.StatusCode(), http.StatusBadRequest)
	}
	if err.Error() != "missing db path" {
		t.Fatalf("got message %q, want missing db path", err.Error())
	}
}

func TestErrorfFormatsMessage(t *testing.T) {
	err := Errorf(NotFound, "run %s not found", "abc")

	if err.Error() != "run abc not found" {
		t.Fatalf("got message %q, want formatted message", err.Error())
	}
}

func TestErrorsIsMatchesByCode(t *testing.T) {
	err := Errorf(IdempotencyConflict, "existing run %s", "abc")

	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("errors.Is did not match idempotency conflict code")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("errors.Is unexpectedly matched not found")
	}
}
