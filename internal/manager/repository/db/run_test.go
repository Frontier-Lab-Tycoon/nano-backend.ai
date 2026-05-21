package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	runspecpreset "github.com/seedspirit/nano-backend.ai/internal/manager/runspec/preset"
)

const (
	testCreatedAt       = "2026-05-21T00:00:00Z"
	testModelOptions    = `{"base_model":"unsloth/Llama-3.1-8B"}`
	testDataOptions     = `{"datasets":[{"path":"mergeowl/v1","split":"train"}]}`
	testResourceOptions = `{"gpu":{"count":1},"memory":{"limit_bytes":34359738368},"timeout":{"duration_seconds":14400}}`
	testTrainingOptions = `{"parameters":{"lora_r":32}}`
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
		WHERE type = 'table' AND name IN (
			'projects', 'specs', 'preset_categories', 'presets', 'trainer_presets',
			'preset_option_rules', 'preset_default_values', 'spec_preset_refs', 'runs', 'artifacts'
		)
	`); err != nil {
		t.Fatalf("failed to inspect sqlite schema: %v", err)
	}
	if count != 10 {
		t.Fatalf("got %d migrated tables, want 10", count)
	}

	var categoryCount int
	if err := repo.db.GetContext(ctx, &categoryCount, `SELECT COUNT(*) FROM preset_categories`); err != nil {
		t.Fatalf("failed to inspect preset categories: %v", err)
	}
	if categoryCount != 3 {
		t.Fatalf("got %d preset categories, want 3", categoryCount)
	}

	var presetCount int
	if err := repo.db.GetContext(ctx, &presetCount, `SELECT COUNT(*) FROM presets`); err != nil {
		t.Fatalf("failed to inspect presets: %v", err)
	}
	if presetCount != 2 {
		t.Fatalf("got %d presets, want 2", presetCount)
	}
}

func TestGetSpecUsesRunID(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()
	specID := fixture.givenSpec(projectID, "mergeowl-exp-42")
	runID := fixture.givenRunForSpec(projectID, specID, testCreatedAt)
	trainerPresetID := runspecpreset.PresetAxolotlLoRASFT

	fixture.givenTrainerPresetRef(specID, trainerPresetID)

	got, err := fixture.repo.GetSpec(fixture.ctx, runID)
	if err != nil {
		t.Fatalf("get spec by run id: %v", err)
	}
	if got.ID != specID {
		t.Fatalf("got spec id %s, want %s", got.ID, specID)
	}
	if got.PresetRefs.Trainer == nil || *got.PresetRefs.Trainer != trainerPresetID {
		t.Fatalf("got trainer preset ref %v, want %s", got.PresetRefs.Trainer, trainerPresetID)
	}

	_, err = fixture.repo.GetSpec(fixture.ctx, specID)
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got err %v, want ErrNotFound when using spec id", err)
	}
}

func TestListProjectRunsReturnsMostRecentRunsWithinLimit(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()
	fixture.givenRun(projectID, "old", "2026-05-21T00:00:00Z")
	middleRunID := fixture.givenRun(projectID, "middle", "2026-05-21T00:01:00Z")
	newRunID := fixture.givenRun(projectID, "new", "2026-05-21T00:02:00Z")

	got, err := fixture.repo.ListProjectRuns(fixture.ctx, projectID, 2)
	if err != nil {
		t.Fatalf("list project runs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d runs, want 2", len(got))
	}
	if got[0].ID != newRunID {
		t.Fatalf("got first run id %s, want %s", got[0].ID, newRunID)
	}
	if got[1].ID != middleRunID {
		t.Fatalf("got second run id %s, want %s", got[1].ID, middleRunID)
	}
	if got[0].Lifecycle.Status != run.Queued {
		t.Fatalf("got status %s, want %s", got[0].Lifecycle.Status, run.Queued)
	}
}

func TestListProjectRunsReturnsEmptyForProjectWithoutRuns(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()

	got, err := fixture.repo.ListProjectRuns(fixture.ctx, projectID, 20)
	if err != nil {
		t.Fatalf("list project runs: %v", err)
	}
	if got == nil {
		t.Fatal("got nil runs, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("got %d runs, want 0", len(got))
	}
}

func TestListProjectRunsReturnsNotFoundForMissingProject(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := uuid.New()

	_, err := fixture.repo.ListProjectRuns(fixture.ctx, projectID, 20)
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got err %v, want ErrNotFound", err)
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

type runRepositoryFixture struct {
	t    *testing.T
	ctx  context.Context
	repo *RunRepository
}

func newRunRepositoryFixture(t *testing.T) *runRepositoryFixture {
	t.Helper()
	ctx := context.Background()
	return &runRepositoryFixture{
		t:    t,
		ctx:  ctx,
		repo: newTestRunRepository(t, ctx),
	}
}

func (f *runRepositoryFixture) givenProject() uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	if _, err := f.repo.db.ExecContext(f.ctx, `
		INSERT INTO projects (id, name, description, created_at)
		VALUES (?, ?, ?, ?)
	`, id.String(), "mergeowl", "MergeOwl experiments", testCreatedAt); err != nil {
		f.t.Fatalf("insert project: %v", err)
	}
	return id
}

func (f *runRepositoryFixture) givenSpec(projectID uuid.UUID, name string) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	if _, err := f.repo.db.ExecContext(f.ctx, `
		INSERT INTO specs (
			id, project_id, name, description, model_options, data_options,
			resource_options, training_options, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id.String(), projectID.String(), name, "LoRA SFT experiment",
		testModelOptions, testDataOptions, testResourceOptions, testTrainingOptions, testCreatedAt); err != nil {
		f.t.Fatalf("insert spec: %v", err)
	}
	return id
}

func (f *runRepositoryFixture) givenRun(projectID uuid.UUID, name, createdAt string) uuid.UUID {
	f.t.Helper()
	specID := f.givenSpec(projectID, name)
	return f.givenRunForSpec(projectID, specID, createdAt)
}

func (f *runRepositoryFixture) givenRunForSpec(projectID, specID uuid.UUID, createdAt string) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	if _, err := f.repo.db.ExecContext(f.ctx, `
		INSERT INTO runs (id, project_id, spec_id, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id.String(), projectID.String(), specID.String(), "queued", createdAt); err != nil {
		f.t.Fatalf("insert run: %v", err)
	}
	return id
}

func (f *runRepositoryFixture) givenTrainerPresetRef(specID, presetID uuid.UUID) {
	f.t.Helper()
	if _, err := f.repo.db.ExecContext(f.ctx, `
		INSERT INTO spec_preset_refs (spec_id, category, preset_id)
		VALUES (?, ?, ?)
	`, specID.String(), "trainer", presetID.String()); err != nil {
		f.t.Fatalf("insert preset ref: %v", err)
	}
}
