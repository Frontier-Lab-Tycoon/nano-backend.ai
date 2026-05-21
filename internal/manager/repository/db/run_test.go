package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
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
	ctx := context.Background()
	repo := newTestRunRepository(t, ctx)
	projectID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	specID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	runID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	trainerPresetID := runspecpreset.PresetAxolotlLoRASFT

	if _, err := repo.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, description, created_at)
		VALUES (?, ?, ?, ?)
	`, projectID.String(), "mergeowl", "MergeOwl experiments", testCreatedAt); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `
		INSERT INTO specs (
			id, project_id, name, description, model_options, data_options,
			resource_options, training_options, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, specID.String(), projectID.String(), "mergeowl-exp-42", "LoRA SFT experiment",
		testModelOptions, testDataOptions, testResourceOptions, testTrainingOptions, testCreatedAt); err != nil {
		t.Fatalf("insert spec: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `
		INSERT INTO spec_preset_refs (spec_id, category, preset_id)
		VALUES (?, ?, ?)
	`, specID.String(), "trainer", trainerPresetID.String()); err != nil {
		t.Fatalf("insert preset ref: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `
		INSERT INTO runs (id, project_id, spec_id, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, runID.String(), projectID.String(), specID.String(), "queued", testCreatedAt); err != nil {
		t.Fatalf("insert run: %v", err)
	}

	got, err := repo.GetSpec(ctx, runID)
	if err != nil {
		t.Fatalf("get spec by run id: %v", err)
	}
	if got.ID != specID {
		t.Fatalf("got spec id %s, want %s", got.ID, specID)
	}
	if got.PresetRefs.Trainer == nil || *got.PresetRefs.Trainer != trainerPresetID {
		t.Fatalf("got trainer preset ref %v, want %s", got.PresetRefs.Trainer, trainerPresetID)
	}

	_, err = repo.GetSpec(ctx, specID)
	if !errors.Is(err, errordef.ErrNotFound) {
		t.Fatalf("got err %v, want ErrNotFound when using spec id", err)
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
