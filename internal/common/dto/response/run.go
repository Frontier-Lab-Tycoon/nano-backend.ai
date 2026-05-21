package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
)

// RunSummary is the stable list item shape for run navigation responses.
type RunSummary struct {
	ID             uuid.UUID          `json:"id"`
	ProjectID      uuid.UUID          `json:"project_id"`
	SpecID         uuid.UUID          `json:"spec_id"`
	IdempotencyKey *string            `json:"idempotency_key,omitempty"`
	Status         run.Status         `json:"status"`
	FailureReason  *run.FailureReason `json:"failure_reason,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	StartedAt      *time.Time         `json:"started_at,omitempty"`
	FinishedAt     *time.Time         `json:"finished_at,omitempty"`
}

// ProjectRunsData is the response data payload for project run lists.
type ProjectRunsData struct {
	Runs  []RunSummary `json:"runs"`
	Limit int          `json:"limit"`
}

// NewRunSummary converts application run data into the external summary DTO.
func NewRunSummary(projectID uuid.UUID, source *run.Run) RunSummary {
	return RunSummary{
		ID:             source.ID,
		ProjectID:      projectID,
		SpecID:         source.SpecID,
		IdempotencyKey: source.IdempotencyKey,
		Status:         source.Lifecycle.Status,
		FailureReason:  source.Lifecycle.FailureReason,
		CreatedAt:      source.Lifecycle.CreatedAt,
		StartedAt:      source.Lifecycle.StartedAt,
		FinishedAt:     source.Lifecycle.FinishedAt,
	}
}

// NewRunSummaries converts application run data into external summary DTOs.
func NewRunSummaries(projectID uuid.UUID, source []run.Run) []RunSummary {
	summaries := make([]RunSummary, 0, len(source))
	for i := range source {
		summaries = append(summaries, NewRunSummary(projectID, &source[i]))
	}
	return summaries
}
