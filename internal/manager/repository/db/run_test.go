package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/project"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
)

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)

	if err := Migrate(ctx, repo.db); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}

	var count int
	if err := repo.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'table' AND name IN ('projects', 'specs', 'runs', 'artifacts')
	`); err != nil {
		t.Fatalf("failed to inspect sqlite schema: %v", err)
	}
	if count != 4 {
		t.Fatalf("got %d migrated tables, want 4", count)
	}
}

func TestCreateGetAndListRuns(t *testing.T) {
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)
	proj := sampleProject()
	if err := repo.CreateProject(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	firstSpec := sampleSpec(proj.ID)
	secondSpec := sampleSpec(proj.ID)
	secondSpec.Name = "second"
	if err := repo.CreateSpec(ctx, &firstSpec); err != nil {
		t.Fatalf("create first spec: %v", err)
	}
	if err := repo.CreateSpec(ctx, &secondSpec); err != nil {
		t.Fatalf("create second spec: %v", err)
	}

	firstRun := run.NewRun(firstSpec.ID)
	firstRun.Lifecycle.CreatedAt = time.Date(2026, 5, 3, 1, 0, 0, 0, time.UTC)
	secondRun := run.NewRun(secondSpec.ID)
	secondRun.Lifecycle.CreatedAt = time.Date(2026, 5, 3, 2, 0, 0, 0, time.UTC)
	if err := repo.CreateRun(ctx, &firstRun, proj.ID); err != nil {
		t.Fatalf("create first run: %v", err)
	}
	if err := repo.CreateRun(ctx, &secondRun, proj.ID); err != nil {
		t.Fatalf("create second run: %v", err)
	}

	got, err := repo.GetRun(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("get first run: %v", err)
	}
	if got.ID != firstRun.ID || got.SpecID != firstSpec.ID || got.Lifecycle.Status != run.Queued {
		t.Fatalf("got run %+v, want id %s spec %s queued", got, firstRun.ID, firstSpec.ID)
	}

	listed, err := repo.ListRuns(ctx, proj.ID)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("got %d listed runs, want 2", len(listed))
	}
	if listed[0].ID != secondRun.ID || listed[1].ID != firstRun.ID {
		t.Fatalf("runs are not ordered by newest created_at: got %s then %s", listed[0].ID, listed[1].ID)
	}
}

func TestUpdateLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)
	proj := sampleProject()
	if err := repo.CreateProject(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}
	rn := submitSampleRun(t, ctx, repo, proj.ID, nil)

	startedAt := time.Date(2026, 5, 3, 3, 0, 0, 0, time.UTC)
	if err := rn.Transition(run.Next(run.Preparing), startedAt); err != nil {
		t.Fatalf("transition to preparing: %v", err)
	}
	if err := rn.Transition(run.Fail(run.FailureReason("dataset_stage_failed")), startedAt.Add(time.Minute)); err != nil {
		t.Fatalf("transition to failed: %v", err)
	}
	if err := repo.UpdateLifecycle(ctx, rn.ID, rn.Lifecycle); err != nil {
		t.Fatalf("update lifecycle: %v", err)
	}

	got, err := repo.GetRun(ctx, rn.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Lifecycle.Status != run.Failed {
		t.Fatalf("got status %q, want %q", got.Lifecycle.Status, run.Failed)
	}
	if got.Lifecycle.FailureReason == nil || *got.Lifecycle.FailureReason != "dataset_stage_failed" {
		t.Fatalf("got failure reason %v, want dataset_stage_failed", got.Lifecycle.FailureReason)
	}
	if got.Lifecycle.FinishedAt == nil || !got.Lifecycle.FinishedAt.Equal(startedAt.Add(time.Minute)) {
		t.Fatalf("got finished_at %v, want %v", got.Lifecycle.FinishedAt, startedAt.Add(time.Minute))
	}
	if !got.Lifecycle.CreatedAt.Equal(rn.Lifecycle.CreatedAt) {
		t.Fatalf("got created_at %v, want unchanged %v", got.Lifecycle.CreatedAt, rn.Lifecycle.CreatedAt)
	}
}

func TestArtifactIndexRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)
	proj := sampleProject()
	if err := repo.CreateProject(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}
	rn := submitSampleRun(t, ctx, repo, proj.ID, nil)

	index := run.NewArtifactIndex(rn.ID, "/artifacts/project/run")
	index.Files = []run.ArtifactFile{
		{Path: "metrics.json", SizeBytes: 123, SHA256: "abc"},
		{Path: "report.md", SizeBytes: 456, SHA256: "def"},
	}
	if err := repo.SaveArtifactIndex(ctx, index); err != nil {
		t.Fatalf("save artifact index: %v", err)
	}

	got, err := repo.GetArtifactIndex(ctx, rn.ID)
	if err != nil {
		t.Fatalf("get artifact index: %v", err)
	}
	if got.RunID != index.RunID || got.BasePath != index.BasePath {
		t.Fatalf("got artifact index %+v, want run %s base path %s", got, index.RunID, index.BasePath)
	}
	if len(got.Files) != 2 || got.Files[0].Path != "metrics.json" || got.Files[1].Path != "report.md" {
		t.Fatalf("got artifact files %+v, want metrics.json and report.md", got.Files)
	}

	var ids []string
	if err := repo.db.SelectContext(ctx, &ids, `SELECT id FROM artifacts ORDER BY path ASC`); err != nil {
		t.Fatalf("select artifact ids: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d artifact ids, want 2", len(ids))
	}
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			t.Fatalf("artifact id %q is not a UUID: %v", id, err)
		}
	}

	var artifactCreatedAts []string
	if err := repo.db.SelectContext(ctx, &artifactCreatedAts, `SELECT created_at FROM artifacts ORDER BY path ASC`); err != nil {
		t.Fatalf("select artifact created_at values: %v", err)
	}
	for _, createdAt := range artifactCreatedAts {
		if _, err := time.Parse(time.RFC3339Nano, createdAt); err != nil {
			t.Fatalf("artifact created_at %q is not RFC3339Nano: %v", createdAt, err)
		}
	}
}

func TestSubmitRunIdempotencyExactMatch(t *testing.T) {
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)
	proj := sampleProject()
	if err := repo.CreateProject(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	key := "mergeowl-exp-42"
	spec := sampleSpec(proj.ID)
	first, err := repo.SubmitRun(ctx, &spec, &key)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}

	retry := spec
	retry.ID = uuid.New()
	second, err := repo.SubmitRun(ctx, &retry, &key)
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("got second run %s, want existing run %s", second.ID, first.ID)
	}
}

func TestSubmitRunIdempotencyConflict(t *testing.T) {
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)
	proj := sampleProject()
	if err := repo.CreateProject(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	key := "mergeowl-exp-42"
	spec := sampleSpec(proj.ID)
	first, err := repo.SubmitRun(ctx, &spec, &key)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	conflictingSpec := sampleSpec(proj.ID)
	conflictingSpec.Name = "different-intent"

	_, err = repo.SubmitRun(ctx, &conflictingSpec, &key)
	if !errors.Is(err, errordef.ErrIdempotencyConflict) {
		t.Fatalf("got err %v, want ErrIdempotencyConflict", err)
	}
	var conflict *IdempotencyConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("got err %T, want IdempotencyConflictError", err)
	}
	if conflict.ExistingRunID != first.ID {
		t.Fatalf("got existing run %s, want %s", conflict.ExistingRunID, first.ID)
	}
}

func newTestRunRepository(t *testing.T, ctx context.Context) *RunRepository {
	t.Helper()
	repo, err := NewRunRepository(ctx, Args{
		DBPath: filepath.Join(t.TempDir(), "manager.db"),
	})
	if err != nil {
		t.Fatalf("new run repository: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("close run repository: %v", err)
		}
	})
	return repo
}

func submitSampleRun(t *testing.T, ctx context.Context, repo *RunRepository, projectID uuid.UUID, key *string) run.Run {
	t.Helper()
	spec := sampleSpec(projectID)
	rn, err := repo.SubmitRun(ctx, &spec, key)
	if err != nil {
		t.Fatalf("submit sample run: %v", err)
	}
	return rn
}

func sampleProject() project.Project {
	p := project.New("mergeowl", "MergeOwl experiments")
	p.CreatedAt = time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	return p
}

func sampleSpec(projectID uuid.UUID) run.Spec {
	spec := run.NewSpec(projectID, "mergeowl-exp-42")
	spec.Description = "LoRA SFT experiment"
	spec.ModelOptions = run.ModelOptions{
		BaseModel: "unsloth/Llama-3.1-8B",
	}
	spec.DataOptions = run.DataOptions{
		Datasets: []run.DatasetRef{
			{Path: "mergeowl/v1", Split: "train"},
		},
	}
	spec.ResourceOptions = run.ResourceOptions{
		GPU:     run.GPUOptions{Count: 1},
		Memory:  run.MemoryOptions{LimitBytes: 34359738368},
		Timeout: run.TimeoutOptions{DurationSeconds: 14400},
	}
	spec.TrainingOptions = run.TrainingOptions{
		Overrides: map[string]any{
			"learning_rate":  2.0e-4,
			"lora_r":         32,
			"max_seq_length": 4096,
			"num_epochs":     3,
			"preset":         "axolotl-lora-sft",
		},
	}
	return spec
}
