package db

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	runspecpreset "github.com/seedspirit/nano-backend.ai/internal/manager/runspec/preset"
)

const (
	testCreatedAt           = "2026-05-21T00:00:00Z"
	testBaseModel           = "unsloth/Llama-3.1-8B"
	testGPUCount            = 1
	testMemoryLimitBytes    = int64(34359738368)
	testTimeoutDurationSecs = int64(14400)
	testDatasetRef          = "mergeowl/v1"
	testSplitName           = "train"
	testTrainingParamKey    = "lora_r"
	testTrainingParamValue  = "32"
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
			'projects', 'specs', 'spec_datasets', 'spec_training_parameters',
			'preset_categories', 'presets', 'trainer_presets',
			'preset_option_rules', 'preset_default_values', 'spec_preset_refs',
			'runs', 'artifacts'
		)
	`); err != nil {
		t.Fatalf("failed to inspect sqlite schema: %v", err)
	}
	if count != 12 {
		t.Fatalf("got %d migrated tables, want 12", count)
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
	if got.ModelOptions.BaseModel != testBaseModel {
		t.Fatalf("got base model %q, want %q", got.ModelOptions.BaseModel, testBaseModel)
	}
	if got.ResourceOptions.GPU.Count != testGPUCount {
		t.Fatalf("got gpu count %d, want %d", got.ResourceOptions.GPU.Count, testGPUCount)
	}
	if got.ResourceOptions.Memory.LimitBytes != testMemoryLimitBytes {
		t.Fatalf("got memory limit %d, want %d", got.ResourceOptions.Memory.LimitBytes, testMemoryLimitBytes)
	}
	if got.ResourceOptions.Timeout.DurationSeconds != testTimeoutDurationSecs {
		t.Fatalf("got timeout %d, want %d", got.ResourceOptions.Timeout.DurationSeconds, testTimeoutDurationSecs)
	}
	if len(got.DataOptions.Datasets) != 1 {
		t.Fatalf("got %d datasets, want 1", len(got.DataOptions.Datasets))
	}
	if got.DataOptions.Datasets[0].Path != testDatasetRef {
		t.Fatalf("got dataset path %q, want %q", got.DataOptions.Datasets[0].Path, testDatasetRef)
	}
	if got.DataOptions.Datasets[0].Split != testSplitName {
		t.Fatalf("got dataset split %q, want %q", got.DataOptions.Datasets[0].Split, testSplitName)
	}
	gotParam, ok := got.TrainingOptions.Parameters[testTrainingParamKey].(json.Number)
	if !ok {
		t.Fatalf("got training param %v (%T), want json.Number", got.TrainingOptions.Parameters[testTrainingParamKey], got.TrainingOptions.Parameters[testTrainingParamKey])
	}
	if string(gotParam) != testTrainingParamValue {
		t.Fatalf("got training param value %q, want %q", gotParam, testTrainingParamValue)
	}
	intValue, err := gotParam.Int64()
	if err != nil {
		t.Fatalf("training param Int64 cast: %v", err)
	}
	if intValue != 32 {
		t.Fatalf("got training param int value %d, want 32", intValue)
	}

	_, err = fixture.repo.GetSpec(fixture.ctx, specID)
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got err %v, want ErrNotFound when using spec id", err)
	}
}

func TestGetSpecPreservesFloatTrainingParameter(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()
	specID := fixture.givenSpec(projectID, "mergeowl-exp-float")
	runID := fixture.givenRunForSpec(projectID, specID, testCreatedAt)

	if _, err := fixture.repo.db.ExecContext(fixture.ctx, `
		INSERT INTO spec_training_parameters (spec_id, key, value)
		VALUES (?, ?, ?)
	`, specID.String(), "learning_rate", "0.0002"); err != nil {
		t.Fatalf("insert float training parameter: %v", err)
	}

	got, err := fixture.repo.GetSpec(fixture.ctx, runID)
	if err != nil {
		t.Fatalf("get spec: %v", err)
	}
	gotValue, ok := got.TrainingOptions.Parameters["learning_rate"].(json.Number)
	if !ok {
		t.Fatalf("got learning_rate %v (%T), want json.Number", got.TrainingOptions.Parameters["learning_rate"], got.TrainingOptions.Parameters["learning_rate"])
	}
	if string(gotValue) != "0.0002" {
		t.Fatalf("got learning_rate literal %q, want %q", gotValue, "0.0002")
	}
	floatValue, err := gotValue.Float64()
	if err != nil {
		t.Fatalf("learning_rate Float64 cast: %v", err)
	}
	if floatValue != 0.0002 {
		t.Fatalf("got learning_rate float %v, want 0.0002", floatValue)
	}
}

func TestSpecChildRowsCascadeOnDelete(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()
	specID := fixture.givenSpec(projectID, "mergeowl-exp-cascade")

	if _, err := fixture.repo.db.ExecContext(fixture.ctx, `DELETE FROM specs WHERE id = ?`, specID.String()); err != nil {
		t.Fatalf("delete spec: %v", err)
	}

	var datasetCount int
	if err := fixture.repo.db.GetContext(fixture.ctx, &datasetCount, `
		SELECT COUNT(*) FROM spec_datasets WHERE spec_id = ?
	`, specID.String()); err != nil {
		t.Fatalf("count spec_datasets: %v", err)
	}
	if datasetCount != 0 {
		t.Fatalf("got %d spec_datasets rows after CASCADE, want 0", datasetCount)
	}

	var parameterCount int
	if err := fixture.repo.db.GetContext(fixture.ctx, &parameterCount, `
		SELECT COUNT(*) FROM spec_training_parameters WHERE spec_id = ?
	`, specID.String()); err != nil {
		t.Fatalf("count spec_training_parameters: %v", err)
	}
	if parameterCount != 0 {
		t.Fatalf("got %d spec_training_parameters rows after CASCADE, want 0", parameterCount)
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
			id, project_id, name, description,
			model_base_model,
			resource_cpu_cores, resource_gpu_count,
			resource_memory_limit_bytes, resource_timeout_duration_seconds,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id.String(), projectID.String(), name, "LoRA SFT experiment",
		testBaseModel,
		0, testGPUCount,
		testMemoryLimitBytes, testTimeoutDurationSecs,
		testCreatedAt); err != nil {
		f.t.Fatalf("insert spec: %v", err)
	}
	if _, err := f.repo.db.ExecContext(f.ctx, `
		INSERT INTO spec_datasets (spec_id, ordinal, dataset_ref, split_name)
		VALUES (?, ?, ?, ?)
	`, id.String(), 0, testDatasetRef, testSplitName); err != nil {
		f.t.Fatalf("insert spec dataset: %v", err)
	}
	if _, err := f.repo.db.ExecContext(f.ctx, `
		INSERT INTO spec_training_parameters (spec_id, key, value)
		VALUES (?, ?, ?)
	`, id.String(), testTrainingParamKey, testTrainingParamValue); err != nil {
		f.t.Fatalf("insert spec training parameter: %v", err)
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

func TestCreateRunPersistsSpecAndRun(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()
	specValue := sampleSpec(projectID)
	runValue := run.NewWithSpec(specValue.ID, projectID)

	if err := fixture.repo.CreateRun(fixture.ctx, &specValue, &runValue); err != nil {
		t.Fatalf("create run: %v", err)
	}

	gotSpec, err := fixture.repo.GetSpec(fixture.ctx, runValue.ID)
	if err != nil {
		t.Fatalf("get spec by run id: %v", err)
	}
	if gotSpec.ID != specValue.ID {
		t.Fatalf("got spec id %s, want %s", gotSpec.ID, specValue.ID)
	}
	if gotSpec.ModelOptions.BaseModel != "meta-llama/Llama-3-8B" {
		t.Fatalf("got base model %q", gotSpec.ModelOptions.BaseModel)
	}
}

func TestCreateRunRollsBackOnProjectFKViolation(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	missingProjectID := uuid.New()
	specValue := sampleSpec(missingProjectID)
	runValue := run.NewWithSpec(specValue.ID, missingProjectID)

	err := fixture.repo.CreateRun(fixture.ctx, &specValue, &runValue)
	if err == nil {
		t.Fatal("got nil error, want FK violation")
	}

	for _, kv := range []struct {
		query string
		arg   string
	}{
		{`SELECT COUNT(*) FROM specs WHERE id = ?`, specValue.ID.String()},
		{`SELECT COUNT(*) FROM runs WHERE id = ?`, runValue.ID.String()},
	} {
		var n int
		if err := fixture.repo.db.GetContext(fixture.ctx, &n, kv.query, kv.arg); err != nil {
			t.Fatalf("count %q: %v", kv.query, err)
		}
		if n != 0 {
			t.Fatalf("query %q got %d rows after rollback, want 0", kv.query, n)
		}
	}
}

func TestProjectExistsReportsPresence(t *testing.T) {
	fixture := newRunRepositoryFixture(t)
	projectID := fixture.givenProject()

	if err := fixture.repo.ProjectExists(fixture.ctx, projectID); err != nil {
		t.Fatalf("existing project: %v", err)
	}
	if err := fixture.repo.ProjectExists(fixture.ctx, uuid.New()); !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("missing project got err %v, want ErrNotFound", err)
	}
}

func sampleSpec(projectID uuid.UUID) spec.Spec {
	specID := uuid.New()
	trainerID := runspecpreset.PresetAxolotlLoRASFT
	return spec.Spec{
		ID:           specID,
		ProjectID:    projectID,
		Name:         "submission-1",
		Description:  "",
		PresetRefs:   preset.Refs{Trainer: &trainerID},
		ModelOptions: spec.ModelOptions{BaseModel: "meta-llama/Llama-3-8B"},
		DataOptions: spec.DataOptions{
			Datasets: []spec.DatasetRef{{Path: "tatsu-lab/alpaca", Split: "train"}},
		},
		ResourceOptions: spec.ResourceOptions{
			GPU:     run.GPUOptions{Count: 1},
			Memory:  run.MemoryOptions{LimitBytes: 1 << 30},
			Timeout: run.TimeoutOptions{DurationSeconds: 3600},
		},
		TrainingOptions: spec.TrainingOptions{
			Parameters: map[string]any{
				"learning_rate": 0.0002,
				"num_epochs":    3,
			},
		},
	}
}
