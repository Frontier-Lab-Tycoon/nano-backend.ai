package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/preset"
	"github.com/seedspirit/nano-backend.ai/internal/common/data/run/spec"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	"github.com/seedspirit/nano-backend.ai/internal/manager/repository/db/entity"
)

// Args configures the SQLite run repository.
type Args struct {
	DBPath string
}

// RunRepository reads run ledger data from SQLite.
type RunRepository struct {
	db *sqlx.DB
}

// NewRunRepository opens, migrates, and returns a SQLite run repository.
func NewRunRepository(ctx context.Context, args Args) (*RunRepository, error) {
	dbx, err := Open(ctx, args)
	if err != nil {
		return nil, err
	}
	return &RunRepository{db: dbx}, nil
}

// Close releases the repository database handle.
func (r *RunRepository) Close() error {
	return r.db.Close()
}

// GetSpec returns the finalized spec for a run.
func (r *RunRepository) GetSpec(ctx context.Context, runID uuid.UUID) (spec.Spec, error) {
	var row entity.Spec
	err := r.db.GetContext(ctx, &row, `
		SELECT specs.id, specs.project_id, specs.name, specs.description,
			specs.model_base_model,
			specs.resource_cpu_cores, specs.resource_gpu_count,
			specs.resource_memory_limit_bytes, specs.resource_timeout_duration_seconds,
			specs.created_at
		FROM runs
		JOIN specs ON specs.id = runs.spec_id
		WHERE runs.id = ?
	`, runID.String())
	if errors.Is(err, sql.ErrNoRows) {
		return spec.Spec{}, errordef.ErrNotFound
	}
	if err != nil {
		return spec.Spec{}, fmt.Errorf("get spec for run %s: %w", runID, err)
	}
	specID, err := uuid.Parse(row.ID)
	if err != nil {
		return spec.Spec{}, fmt.Errorf("parse spec id %q: %w", row.ID, err)
	}
	refs, err := r.getSpecPresetRefs(ctx, specID)
	if err != nil {
		return spec.Spec{}, err
	}
	row.PresetRefs = refs

	datasets, err := r.getSpecDatasets(ctx, specID)
	if err != nil {
		return spec.Spec{}, err
	}
	row.Datasets = datasets

	parameters, err := r.getSpecTrainingParameters(ctx, specID)
	if err != nil {
		return spec.Spec{}, err
	}
	row.TrainingParameters = parameters

	return row.ToSpec()
}

// ListProjectRuns returns the most recent runs for a project.
func (r *RunRepository) ListProjectRuns(ctx context.Context, projectID uuid.UUID, limit int) ([]run.Run, error) {
	var exists int
	err := r.db.GetContext(ctx, &exists, `
		SELECT 1
		FROM projects
		WHERE id = ?
	`, projectID.String())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errordef.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check project %s exists: %w", projectID, err)
	}

	var rows []entity.Run
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT id, spec_id, idempotency_key, status, failure_reason,
			created_at, started_at, finished_at
		FROM runs
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, projectID.String(), limit); err != nil {
		return nil, fmt.Errorf("list runs for project %s: %w", projectID, err)
	}

	runs := make([]run.Run, 0, len(rows))
	for i := range rows {
		item, err := rows[i].ToRun()
		if err != nil {
			return nil, err
		}
		runs = append(runs, item)
	}
	return runs, nil
}

func (r *RunRepository) getSpecDatasets(ctx context.Context, specID uuid.UUID) ([]entity.SpecDataset, error) {
	var rows []entity.SpecDataset
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT ordinal, dataset_ref, split_name
		FROM spec_datasets
		WHERE spec_id = ?
		ORDER BY ordinal
	`, specID.String()); err != nil {
		return nil, fmt.Errorf("get spec datasets %s: %w", specID, err)
	}
	return rows, nil
}

func (r *RunRepository) getSpecTrainingParameters(ctx context.Context, specID uuid.UUID) ([]entity.SpecTrainingParameter, error) {
	var rows []entity.SpecTrainingParameter
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT key, value
		FROM spec_training_parameters
		WHERE spec_id = ?
	`, specID.String()); err != nil {
		return nil, fmt.Errorf("get spec training parameters %s: %w", specID, err)
	}
	return rows, nil
}

func (r *RunRepository) getSpecPresetRefs(ctx context.Context, specID uuid.UUID) (preset.Refs, error) {
	var rows []struct {
		Category string `db:"category"`
		PresetID string `db:"preset_id"`
	}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT category, preset_id
		FROM spec_preset_refs
		WHERE spec_id = ?
	`, specID.String()); err != nil {
		return preset.Refs{}, fmt.Errorf("get spec preset refs %s: %w", specID, err)
	}

	var refs preset.Refs
	for _, row := range rows {
		id, err := uuid.Parse(row.PresetID)
		if err != nil {
			return preset.Refs{}, fmt.Errorf("parse preset id %q: %w", row.PresetID, err)
		}
		switch preset.Category(row.Category) {
		case preset.TrainerPreset:
			refs.Trainer = &id
		case preset.ResourcePreset:
			refs.Resource = &id
		case preset.OutputPreset:
			refs.Output = &id
		default:
			return preset.Refs{}, fmt.Errorf("unknown preset category %q", row.Category)
		}
	}
	return refs, nil
}
