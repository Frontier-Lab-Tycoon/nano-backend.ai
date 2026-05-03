package run

import (
	"errors"
	"testing"
	"time"
)

func TestStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{name: "queued to preparing", from: Queued, to: Preparing, want: true},
		{name: "preparing to running", from: Preparing, to: Running, want: true},
		{name: "preparing to failed", from: Preparing, to: Failed, want: true},
		{name: "running to succeeded", from: Running, to: Succeeded, want: true},
		{name: "running to failed", from: Running, to: Failed, want: true},
		{name: "queued to running", from: Queued, to: Running, want: false},
		{name: "succeeded is terminal", from: Succeeded, to: Failed, want: false},
		{name: "failed is terminal", from: Failed, to: Running, want: false},
		{name: "unknown status", from: Status("unknown"), to: Queued, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Fatalf("CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLifecycleTransitionTo(t *testing.T) {
	createdAt := time.Date(2026, 5, 2, 1, 2, 3, 0, time.UTC)
	startedAt := createdAt.Add(time.Minute)
	finishedAt := createdAt.Add(2 * time.Minute)
	lifecycle := NewLifecycle(createdAt)

	if err := lifecycle.Transition(Next(Preparing), createdAt.Add(time.Second)); err != nil {
		t.Fatalf("Transition(Preparing) error = %v", err)
	}
	if err := lifecycle.Transition(Next(Running), startedAt); err != nil {
		t.Fatalf("Transition(Running) error = %v", err)
	}

	if lifecycle.Status != Running {
		t.Fatalf("status = %q, want %q", lifecycle.Status, Running)
	}
	if lifecycle.StartedAt == nil || !lifecycle.StartedAt.Equal(startedAt) {
		t.Fatalf("started_at = %v, want %v", lifecycle.StartedAt, startedAt)
	}
	if lifecycle.FinishedAt != nil {
		t.Fatalf("finished_at = %v, want nil", lifecycle.FinishedAt)
	}

	if err := lifecycle.Transition(Fail("trainer_error"), finishedAt); err != nil {
		t.Fatalf("Transition(Failed) error = %v", err)
	}

	if lifecycle.Status != Failed {
		t.Fatalf("status = %q, want %q", lifecycle.Status, Failed)
	}
	if lifecycle.FailureReason == nil || *lifecycle.FailureReason != "trainer_error" {
		t.Fatalf("failure_reason = %v, want trainer_error", lifecycle.FailureReason)
	}
	if lifecycle.FinishedAt == nil || !lifecycle.FinishedAt.Equal(finishedAt) {
		t.Fatalf("finished_at = %v, want %v", lifecycle.FinishedAt, finishedAt)
	}
}

func TestLifecycleTransitionErrors(t *testing.T) {
	now := time.Date(2026, 5, 2, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name       string
		setup      func(*Lifecycle)
		transition Transition
		want       error
	}{
		{
			name: "failed requires reason",
			setup: func(l *Lifecycle) {
				if err := l.Transition(Next(Preparing), now); err != nil {
					t.Fatal(err)
				}
			},
			transition: Next(Failed),
			want:       ErrFailureReasonRequired,
		},
		{
			name: "terminal state cannot transition",
			setup: func(l *Lifecycle) {
				if err := l.Transition(Next(Preparing), now); err != nil {
					t.Fatal(err)
				}
				if err := l.Transition(Next(Running), now); err != nil {
					t.Fatal(err)
				}
				if err := l.Transition(Next(Succeeded), now); err != nil {
					t.Fatal(err)
				}
			},
			transition: Fail("trainer_error"),
			want:       ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifecycle := NewLifecycle(now)
			if tt.setup != nil {
				tt.setup(&lifecycle)
			}

			err := lifecycle.Transition(tt.transition, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}
