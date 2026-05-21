package run

import (
	"errors"
	"time"
)

// Status represents the lifecycle stage of a Run.
//
//	queued → preparing → running → succeeded
//	                   ↓
//	                 failed
type Status string

// Run status constants.
const (
	Queued    Status = "queued"
	Preparing Status = "preparing"
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
)

// FailureReason is the machine-readable reason a Run failed.
type FailureReason string

var (
	// ErrInvalidTransition indicates a status transition is not allowed.
	ErrInvalidTransition = errors.New("invalid run status transition")
	// ErrFailureReasonRequired indicates a failed transition lacks a reason.
	ErrFailureReasonRequired = errors.New("failure reason is required")
)

// Transition describes a requested lifecycle transition.
type Transition struct {
	next          Status
	failureReason *FailureReason
}

// Next creates an ordinary lifecycle transition.
func Next(status Status) Transition {
	return Transition{next: status}
}

// Fail creates a failed lifecycle transition with a machine-readable reason.
func Fail(reason FailureReason) Transition {
	return Transition{
		next:          Failed,
		failureReason: &reason,
	}
}

// Lifecycle groups mutable execution state for a Run.
type Lifecycle struct {
	Status        Status         `json:"status"`
	FailureReason *FailureReason `json:"failure_reason,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty"`
}

// NewLifecycle creates a queued lifecycle with the given creation time.
func NewLifecycle(createdAt time.Time) Lifecycle {
	return Lifecycle{
		Status:    Queued,
		CreatedAt: createdAt,
	}
}

// CanTransitionTo reports whether moving from s to next is a valid state
func (s Status) CanTransitionTo(next Status) bool {
	switch s {
	case Queued:
		return next == Preparing
	case Preparing:
		return next == Running || next == Failed
	case Running:
		return next == Succeeded || next == Failed
	case Succeeded, Failed:
		return false
	default:
		return false
	}
}

// Transition applies the requested lifecycle transition.
func (l *Lifecycle) Transition(transition Transition, at time.Time) error {
	next := transition.next
	if !l.Status.CanTransitionTo(next) {
		return ErrInvalidTransition
	}
	if next == Failed {
		if transition.failureReason == nil || *transition.failureReason == "" {
			return ErrFailureReasonRequired
		}
		reasonCopy := *transition.failureReason
		l.FailureReason = &reasonCopy
	} else {
		l.FailureReason = nil
	}

	l.Status = next
	switch next {
	case Running:
		if l.StartedAt == nil {
			startedAt := at
			l.StartedAt = &startedAt
		}
	case Succeeded, Failed:
		finishedAt := at
		l.FinishedAt = &finishedAt
	}

	return nil
}
