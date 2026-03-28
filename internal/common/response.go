package common

// APIStatus represents the outcome status of an API response.
type APIStatus string

// API status constants.
const (
	StatusOK    APIStatus = "ok"
	StatusError APIStatus = "error"
)

// APIResponse is the standard API response envelope.
// All external API responses use this structure:
//
//	{"status": "...", "reason": "...", "next_action_hint": "..."}
type APIResponse struct {
	Status         APIStatus `json:"status"`
	Reason         string    `json:"reason"`
	NextActionHint string    `json:"next_action_hint"`
}

// NewAPIResponse creates an APIResponse with all fields set.
func NewAPIResponse(status APIStatus, reason, nextActionHint string) APIResponse {
	return APIResponse{
		Status:         status,
		Reason:         reason,
		NextActionHint: nextActionHint,
	}
}

// OKResponse creates a successful APIResponse with status "ok".
func OKResponse(reason, nextActionHint string) APIResponse {
	return NewAPIResponse(StatusOK, reason, nextActionHint)
}

// ErrorResponse creates an error APIResponse with status "error".
func ErrorResponse(reason, nextActionHint string) APIResponse {
	return NewAPIResponse(StatusError, reason, nextActionHint)
}
