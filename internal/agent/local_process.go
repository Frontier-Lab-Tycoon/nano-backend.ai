package agent

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/seedspirit/nano-backend.ai/internal/common/kernel"
)

// Compile-time verification that LocalProcess implements kernel.Runtime.
var _ kernel.Runtime = (*LocalProcess)(nil)

// processEntry tracks a running child process.
type processEntry struct {
	cmd  *exec.Cmd
	done chan struct{} // closed when cmd.Wait() returns
}

// LocalProcess implements kernel.Runtime by managing local OS processes.
type LocalProcess struct {
	mu        sync.Mutex
	processes map[kernel.ID]*processEntry
}

// NewLocalProcess creates a new LocalProcess runtime.
func NewLocalProcess() *LocalProcess {
	return &LocalProcess{
		processes: make(map[kernel.ID]*processEntry),
	}
}

// Create launches a child process described by spec and returns its kernel ID.
func (lp *LocalProcess) Create(ctx context.Context, spec kernel.Spec) (kernel.ID, error) {
	if len(spec.Command) == 0 {
		return kernel.ID{}, &kernel.Error{
			Op:  "create",
			Err: fmt.Errorf("empty command: %w", kernel.ErrRuntime),
		}
	}

	cmd := exec.CommandContext(ctx, spec.Command[0], spec.Command[1:]...) //nolint:gosec // command comes from trusted kernel.Spec
	if err := cmd.Start(); err != nil {
		return kernel.ID{}, &kernel.Error{
			Op:  "create",
			Err: fmt.Errorf("%s: %w", err, kernel.ErrRuntime),
		}
	}

	id := kernel.NewID()
	entry := &processEntry{
		cmd:  cmd,
		done: make(chan struct{}),
	}

	// Reap the process in the background to prevent zombies.
	go func() {
		_ = cmd.Wait()
		close(entry.done)
	}()

	lp.mu.Lock()
	lp.processes[id] = entry
	lp.mu.Unlock()

	return id, nil
}

// Destroy terminates the process identified by id.
func (lp *LocalProcess) Destroy(_ context.Context, id kernel.ID) error {
	lp.mu.Lock()
	entry, ok := lp.processes[id]
	if !ok {
		lp.mu.Unlock()
		return &kernel.Error{
			Op:  "destroy",
			ID:  id,
			Err: kernel.ErrNotFound,
		}
	}
	lp.mu.Unlock()

	if err := entry.cmd.Process.Kill(); err != nil {
		// Process may have already exited; not an error.
		select {
		case <-entry.done:
			lp.mu.Lock()
			delete(lp.processes, id)
			lp.mu.Unlock()
			return nil
		default:
			return &kernel.Error{
				Op:  "destroy",
				ID:  id,
				Err: fmt.Errorf("%s: %w", err, kernel.ErrRuntime),
			}
		}
	}

	// Wait for the reaper goroutine to finish.
	<-entry.done
	lp.mu.Lock()
	delete(lp.processes, id)
	lp.mu.Unlock()
	return nil
}

// Status returns the current status of the process identified by id.
func (lp *LocalProcess) Status(_ context.Context, id kernel.ID) (kernel.Status, error) {
	lp.mu.Lock()
	entry, ok := lp.processes[id]
	lp.mu.Unlock()

	if !ok {
		return kernel.Status{}, &kernel.Error{
			Op:  "status",
			ID:  id,
			Err: kernel.ErrNotFound,
		}
	}

	select {
	case <-entry.done:
		// Process has exited; ProcessState is populated by cmd.Wait().
		return kernel.Exited(entry.cmd.ProcessState.ExitCode()), nil
	default:
		return kernel.Running(), nil
	}
}
