package errordef

import (
	"fmt"
	"net/http"
)

// ErrorCode identifies a machine-readable manager error class.
type ErrorCode struct {
	code       string
	statusCode int
}

type err struct {
	errCode ErrorCode
	message string
}

var (
	// NotFound indicates that a requested resource does not exist.
	NotFound = ErrorCode{code: "not_found", statusCode: http.StatusNotFound}
	// IdempotencyConflict indicates that an idempotency key was reused with different input.
	IdempotencyConflict = ErrorCode{code: "idempotency_conflict", statusCode: http.StatusConflict}
	// ArtifactIndexMissing indicates that a run exists but no artifact index was saved.
	ArtifactIndexMissing = ErrorCode{code: "artifact_index_missing", statusCode: http.StatusNotFound}
	// InvalidInput indicates that a request or internal call passed invalid input.
	InvalidInput = ErrorCode{code: "invalid_input", statusCode: http.StatusBadRequest}
)

var (
	// ErrNotFound is the sentinel error for NotFound.
	ErrNotFound = Error(NotFound, "resource not found")
	// ErrIdempotencyConflict is the sentinel error for IdempotencyConflict.
	ErrIdempotencyConflict = Error(IdempotencyConflict, "idempotency key conflicts with a different spec")
	// ErrArtifactIndexMissing is the sentinel error for ArtifactIndexMissing.
	ErrArtifactIndexMissing = Error(ArtifactIndexMissing, "artifact index not found")
	// ErrInvalidInput is the sentinel error for InvalidInput.
	ErrInvalidInput = Error(InvalidInput, "invalid input")
)

// Error creates an error with a stable machine-readable code.
func Error(code ErrorCode, msg string) error {
	return &err{
		errCode: code,
		message: msg,
	}
}

// Errorf creates a formatted error with a stable machine-readable code.
func Errorf(code ErrorCode, format string, args ...any) error {
	return &err{
		errCode: code,
		message: fmt.Sprintf(format, args...),
	}
}

// StatusCode returns the HTTP status associated with the error code.
func (e *err) StatusCode() int {
	return e.errCode.statusCode
}

// Code returns the machine-readable error code.
func (e *err) Code() string {
	return e.errCode.code
}

func (e *err) Error() string {
	return e.message
}

// Is reports equality by ErrorCode, preserving errors.Is for sentinel errors.
func (e *err) Is(target error) bool {
	targetErr, ok := target.(*err)
	return ok && e.errCode == targetErr.errCode
}
