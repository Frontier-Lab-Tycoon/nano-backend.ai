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
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
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

	return row.ToData()
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
		SELECT id, project_id, spec_id, idempotency_key, status, failure_reason,
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
		item, err := rows[i].ToData()
		if err != nil {
			return nil, err
		}
		runs = append(runs, item)
	}
	return runs, nil
}

// ProjectExists returns nil when the project exists, or errordef.ErrNotFound otherwise.
func (r *RunRepository) ProjectExists(ctx context.Context, projectID uuid.UUID) error {
	var exists int
	err := r.db.GetContext(ctx, &exists, `SELECT 1 FROM projects WHERE id = ?`, projectID.String())
	if errors.Is(err, sql.ErrNoRows) {
		return errordef.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check project %s exists: %w", projectID, err)
	}
	return nil
}

// CreateRun persists a spec and a queued run in a single transaction.
func (r *RunRepository) CreateRun(ctx context.Context, runSpec *spec.Spec, runRecord *run.Run) error {
	createdAt := encoding.FormatTime(runRecord.Lifecycle.CreatedAt)

	specEntity, err := entity.FromData(runSpec, createdAt)
	if err != nil {
		return fmt.Errorf("convert spec: %w", err)
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create-run tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertSpec(ctx, tx, &specEntity); err != nil {
		return err
	}
	if err := insertRun(ctx, tx, runRecord, createdAt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create-run tx: %w", err)
	}
	return nil
}

func insertSpec(ctx context.Context, tx *sqlx.Tx, e *entity.Spec) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO specs (
			id, project_id, name, description,
			model_base_model,
			resource_cpu_cores, resource_gpu_count,
			resource_memory_limit_bytes, resource_timeout_duration_seconds,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.ProjectID, e.Name, e.Description,
		e.ModelBaseModel,
		e.ResourceCPUCores, e.ResourceGPUCount,
		e.ResourceMemoryLimitBytes, e.ResourceTimeoutDurationSeconds,
		e.CreatedAt); err != nil {
		return fmt.Errorf("insert spec %s: %w", e.ID, err)
	}
	for _, ds := range e.Datasets {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO spec_datasets (spec_id, ordinal, dataset_ref, split_name)
			VALUES (?, ?, ?, ?)
		`, e.ID, ds.Ordinal, ds.DatasetRef, ds.SplitName); err != nil {
			return fmt.Errorf("insert spec dataset %s/%d: %w", e.ID, ds.Ordinal, err)
		}
	}
	for _, p := range e.TrainingParameters {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO spec_training_parameters (spec_id, key, value)
			VALUES (?, ?, ?)
		`, e.ID, p.Key, p.Value); err != nil {
			return fmt.Errorf("insert spec parameter %s/%s: %w", e.ID, p.Key, err)
		}
	}
	for _, ref := range []struct {
		category string
		id       *uuid.UUID
	}{
		{string(preset.TrainerPreset), e.PresetRefs.Trainer},
		{string(preset.ResourcePreset), e.PresetRefs.Resource},
		{string(preset.OutputPreset), e.PresetRefs.Output},
	} {
		if ref.id == nil || *ref.id == uuid.Nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO spec_preset_refs (spec_id, category, preset_id)
			VALUES (?, ?, ?)
		`, e.ID, ref.category, ref.id.String()); err != nil {
			return fmt.Errorf("insert spec preset ref %s/%s: %w", e.ID, ref.category, err)
		}
	}
	return nil
}

func insertRun(ctx context.Context, tx *sqlx.Tx, r *run.Run, createdAt string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO runs (
			id, project_id, spec_id, status, created_at
		) VALUES (?, ?, ?, ?, ?)
	`, r.ID.String(), r.ProjectID.String(), r.SpecID.String(),
		string(r.Lifecycle.Status), createdAt); err != nil {
		return fmt.Errorf("insert run %s: %w", r.ID, err)
	}
	return nil
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
