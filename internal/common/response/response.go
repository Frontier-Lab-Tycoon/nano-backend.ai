// Package response defines the standard envelope for external API responses.
package response

import "net/http"

// Error is the standard error payload for external API responses.
//
// Code is the stable machine-readable value clients should branch on. Message
// and NextActionHint are explanatory text for humans and agents.
type Error struct {
	Code           string `json:"code"`
	Message        string `json:"message,omitempty"`
	Details        any    `json:"details,omitempty"`
	NextActionHint string `json:"next_action_hint,omitempty"`
}

// Response is the standard API response envelope.
// Successful responses carry endpoint-specific payloads in Data:
//
//	{"status": 200, "data": {...}}
//
// Error responses carry stable error codes in Error:
//
//	{"status": 404, "error": {"code": "...", "message": "..."}}
type Response struct {
	Status int    `json:"status"`
	Data   any    `json:"data,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// New creates a Response with all fields set.
func New(statusCode int, data any, err *Error) Response {
	return Response{
		Status: statusCode,
		Data:   data,
		Error:  err,
	}
}

// OK creates a successful Response with HTTP 200 status.
func OK(data any) Response {
	return Success(http.StatusOK, data)
}

// Success creates a successful Response with the given HTTP status code.
func Success(statusCode int, data any) Response {
	if data == nil {
		data = map[string]any{}
	}

	return New(statusCode, data, nil)
}

// Err creates an error Response with the given HTTP status code.
func Err(statusCode int, code, message, nextActionHint string, details any) Response {
	return New(statusCode, nil, &Error{
		Code:           code,
		Message:        message,
		Details:        details,
		NextActionHint: nextActionHint,
	})
}
