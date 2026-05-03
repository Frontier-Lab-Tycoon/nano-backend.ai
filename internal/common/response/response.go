// Package response defines the standard envelope for external API responses.
package response

// Status represents the outcome status of an API response.
type Status string

// API status constants.
const (
	StatusOK    Status = "ok"
	StatusError Status = "error"
)

// Response is the standard API response envelope.
// All external API responses use this structure:
//
//	{"status": "...", "reason": "...", "next_action_hint": "..."}
type Response struct {
	Status         Status `json:"status"`
	Reason         string `json:"reason"`
	NextActionHint string `json:"next_action_hint"`
}

// New creates a Response with all fields set.
func New(status Status, reason, nextActionHint string) Response {
	return Response{
		Status:         status,
		Reason:         reason,
		NextActionHint: nextActionHint,
	}
}

// OK creates a successful Response with status "ok".
func OK(reason, nextActionHint string) Response {
	return New(StatusOK, reason, nextActionHint)
}

// Err creates an error Response with status "error".
func Err(reason, nextActionHint string) Response {
	return New(StatusError, reason, nextActionHint)
}
