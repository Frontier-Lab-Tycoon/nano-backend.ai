package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seedspirit/nano-backend.ai/internal/common"
)

func TestCreateSuccess(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = lp.Destroy(ctx, id) })

	if id.IsZero() {
		t.Error("expected non-zero KernelID")
	}
}

func TestCreateReturnsUniqueIDs(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id1, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}
	id2, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error on second create: %v", err)
	}

	if id1.String() == id2.String() {
		t.Errorf("expected unique IDs, got %q and %q", id1, id2)
	}

	t.Cleanup(func() {
		_ = lp.Destroy(ctx, id1)
		_ = lp.Destroy(ctx, id2)
	})
}

func TestCreateEmptyCommand(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	_, err := lp.Create(ctx, common.KernelSpec{Command: []string{}})
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !errors.Is(err, common.ErrKernelRuntime) {
		t.Errorf("expected ErrKernelRuntime, got %v", err)
	}

	var ke *common.KernelError
	if !errors.As(err, &ke) {
		t.Fatalf("expected *KernelError, got %T", err)
	}
	if ke.Op != "create" {
		t.Errorf("expected op %q, got %q", "create", ke.Op)
	}
}

func TestCreateNonexistentBinary(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	_, err := lp.Create(ctx, common.KernelSpec{Command: []string{"nonexistent-binary-xyz-999"}})
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
	if !errors.Is(err, common.ErrKernelRuntime) {
		t.Errorf("expected ErrKernelRuntime, got %v", err)
	}
}

func TestDestroySuccess(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := lp.Destroy(ctx, id); err != nil {
		t.Fatalf("unexpected error on destroy: %v", err)
	}
}

func TestDestroyNotFound(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	fakeID := common.NewKernelID()
	err := lp.Destroy(ctx, fakeID)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
	if !errors.Is(err, common.ErrKernelNotFound) {
		t.Errorf("expected ErrKernelNotFound, got %v", err)
	}

	var ke *common.KernelError
	if !errors.As(err, &ke) {
		t.Fatalf("expected *KernelError, got %T", err)
	}
	if ke.Op != "destroy" {
		t.Errorf("expected op %q, got %q", "destroy", ke.Op)
	}
	if ke.ID.IsZero() {
		t.Error("expected non-zero ID in error")
	}
}

func TestDestroyTwice(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := lp.Destroy(ctx, id); err != nil {
		t.Fatalf("unexpected error on first destroy: %v", err)
	}

	err = lp.Destroy(ctx, id)
	if !errors.Is(err, common.ErrKernelNotFound) {
		t.Errorf("expected ErrKernelNotFound on second destroy, got %v", err)
	}
}

// ── Helpers ───────────────────────────────────────────────────

func waitForExited(t *testing.T, lp *LocalProcess, ctx context.Context, id common.KernelID) common.KernelStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := lp.Status(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Type == common.StatusExited {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout waiting for process to exit")
	return common.KernelStatus{}
}

// ── Status Tests ──────────────────────────────────────────────

func TestStatusRunningProcess(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = lp.Destroy(ctx, id) })

	status, err := lp.Status(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Type != common.StatusRunning {
		t.Errorf("got status %v, want %v", status.Type, common.StatusRunning)
	}
}

func TestStatusExitedZero(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"true"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status := waitForExited(t, lp, ctx, id)
	if status.Code != 0 {
		t.Errorf("got exit code %d, want 0", status.Code)
	}
}

func TestStatusExitedNonZero(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"false"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status := waitForExited(t, lp, ctx, id)
	if status.Code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestStatusNotFound(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	fakeID := common.NewKernelID()
	_, err := lp.Status(ctx, fakeID)
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
	if !errors.Is(err, common.ErrKernelNotFound) {
		t.Errorf("expected ErrKernelNotFound, got %v", err)
	}

	var ke *common.KernelError
	if !errors.As(err, &ke) {
		t.Fatalf("expected *KernelError, got %T", err)
	}
	if ke.Op != "status" {
		t.Errorf("expected op %q, got %q", "status", ke.Op)
	}
	if ke.ID.IsZero() {
		t.Error("expected non-zero ID in error")
	}
}

func TestStatusAfterDestroy(t *testing.T) {
	lp := NewLocalProcess()
	ctx := context.Background()

	id, err := lp.Create(ctx, common.KernelSpec{Command: []string{"sleep", "3600"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := lp.Destroy(ctx, id); err != nil {
		t.Fatalf("unexpected error on destroy: %v", err)
	}

	_, err = lp.Status(ctx, id)
	if !errors.Is(err, common.ErrKernelNotFound) {
		t.Errorf("expected ErrKernelNotFound after destroy, got %v", err)
	}
}
