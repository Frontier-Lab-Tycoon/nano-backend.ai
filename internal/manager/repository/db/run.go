package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
	"github.com/seedspirit/nano-backend.ai/internal/common/project"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	"github.com/seedspirit/nano-backend.ai/internal/manager/repository/db/record"
)

// Args configures the SQLite run repository.
type Args struct {
	DBPath string
}

// RunRepository stores run ledger data in SQLite.
type RunRepository struct {
	db *sqlx.DB
}

// IdempotencyConflictError reports the existing run for a conflicting key.
type IdempotencyConflictError struct {
	ExistingRunID uuid.UUID
}

func (e *IdempotencyConflictError) Error() string {
	return fmt.Sprintf("%v: existing run %s", errordef.ErrIdempotencyConflict, e.ExistingRunID)
}

func (e *IdempotencyConflictError) Unwrap() error {
	return errordef.ErrIdempotencyConflict
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

// CreateProject stores a project.
func (r *RunRepository) CreateProject(ctx context.Context, p project.Project) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, description, created_at)
		VALUES (?, ?, ?, ?)
	`, p.ID.String(), p.Name, p.Description, encoding.FormatTime(p.CreatedAt))
	if err != nil {
		return fmt.Errorf("create project %s: %w", p.ID, err)
	}
	return nil
}

// CreateSpec stores an immutable run spec.
func (r *RunRepository) CreateSpec(ctx context.Context, spec *run.Spec) error {
	if spec == nil {
		return errordef.Errorf(errordef.InvalidInput, "spec is nil")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create spec transaction: %w", err)
	}
	defer rollbackUnlessCommitted(tx)

	if err := insertSpec(ctx, tx, spec); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create spec %s: %w", spec.ID, err)
	}
	return nil
}

// CreateRun stores a run for an existing spec.
func (r *RunRepository) CreateRun(ctx context.Context, rn *run.Run, projectID uuid.UUID) error {
	if rn == nil {
		return errordef.Errorf(errordef.InvalidInput, "run is nil")
	}
	row := record.NewRun(rn, projectID)
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO runs (
			id, project_id, spec_id, idempotency_key, status, failure_reason,
			created_at, started_at, finished_at
		)
		VALUES (
			:id, :project_id, :spec_id, :idempotency_key, :status, :failure_reason,
			:created_at, :started_at, :finished_at
		)
	`, row)
	if err != nil {
		return fmt.Errorf("create run %s: %w", rn.ID, err)
	}
	return nil
}

// SubmitRun creates a spec and queued run, enforcing idempotency when a key is provided.
func (r *RunRepository) SubmitRun(ctx context.Context, spec *run.Spec, idempotencyKey *string) (run.Run, error) {
	if spec == nil {
		return run.Run{}, errordef.Errorf(errordef.InvalidInput, "spec is nil")
	}
	specFingerprint, err := record.ComparableSpecJSON(spec)
	if err != nil {
		return run.Run{}, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return run.Run{}, fmt.Errorf("begin submit run transaction: %w", err)
	}
	defer rollbackUnlessCommitted(tx)

	if idempotencyKey != nil && *idempotencyKey != "" {
		existing, existingFingerprint, err := getRunByIdempotencyKey(ctx, tx, spec.ProjectID, *idempotencyKey)
		if err != nil && !errors.Is(err, errordef.ErrNotFound) {
			return run.Run{}, err
		}
		if err == nil {
			if existingFingerprint == specFingerprint {
				if err := tx.Commit(); err != nil {
					return run.Run{}, fmt.Errorf("commit existing idempotent run lookup: %w", err)
				}
				return existing, nil
			}
			return run.Run{}, &IdempotencyConflictError{ExistingRunID: existing.ID}
		}
	}

	if err := insertSpec(ctx, tx, spec); err != nil {
		return run.Run{}, err
	}

	rn := run.NewRun(spec.ID)
	if idempotencyKey != nil && *idempotencyKey != "" {
		keyCopy := *idempotencyKey
		rn.IdempotencyKey = &keyCopy
	}
	if err := insertRun(ctx, tx, &rn, spec.ProjectID); err != nil {
		return run.Run{}, err
	}
	if err := tx.Commit(); err != nil {
		return run.Run{}, fmt.Errorf("commit submitted run %s: %w", rn.ID, err)
	}

	return rn, nil
}

// GetRun returns a run by ID.
func (r *RunRepository) GetRun(ctx context.Context, id uuid.UUID) (run.Run, error) {
	var row record.Run
	err := r.db.GetContext(ctx, &row, `
		SELECT id, project_id, spec_id, idempotency_key, status, failure_reason,
			artifact_base_path, created_at, started_at, finished_at
		FROM runs
		WHERE id = ?
	`, id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return run.Run{}, errordef.ErrNotFound
	}
	if err != nil {
		return run.Run{}, fmt.Errorf("get run %s: %w", id, err)
	}
	return (&row).ToRun()
}

// ListRuns returns project runs ordered by newest creation time first.
func (r *RunRepository) ListRuns(ctx context.Context, projectID uuid.UUID) ([]run.Run, error) {
	var rows []record.Run
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, project_id, spec_id, idempotency_key, status, failure_reason,
			artifact_base_path, created_at, started_at, finished_at
		FROM runs
		WHERE project_id = ?
		ORDER BY created_at DESC
	`, projectID.String())
	if err != nil {
		return nil, fmt.Errorf("list runs for project %s: %w", projectID, err)
	}

	runs := make([]run.Run, 0, len(rows))
	for i := range rows {
		rn, err := (&rows[i]).ToRun()
		if err != nil {
			return nil, err
		}
		runs = append(runs, rn)
	}
	return runs, nil
}

// UpdateLifecycle persists the mutable lifecycle fields for a run.
func (r *RunRepository) UpdateLifecycle(ctx context.Context, id uuid.UUID, lifecycle run.Lifecycle) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE runs
		SET status = ?, failure_reason = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`, lifecycle.Status, nullFailureReason(lifecycle.FailureReason),
		nullTime(lifecycle.StartedAt), nullTime(lifecycle.FinishedAt), id.String())
	if err != nil {
		return fmt.Errorf("update run lifecycle %s: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check run lifecycle update %s: %w", id, err)
	}
	if affected == 0 {
		return errordef.ErrNotFound
	}
	return nil
}

// SaveArtifactIndex replaces the artifact index metadata for a run.
func (r *RunRepository) SaveArtifactIndex(ctx context.Context, index run.ArtifactIndex) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin artifact index transaction: %w", err)
	}
	defer rollbackUnlessCommitted(tx)

	result, err := tx.ExecContext(ctx, `
		UPDATE runs
		SET artifact_base_path = ?
		WHERE id = ?
	`, index.BasePath, index.RunID.String())
	if err != nil {
		return fmt.Errorf("save artifact base path for run %s: %w", index.RunID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check artifact base path update %s: %w", index.RunID, err)
	}
	if affected == 0 {
		return errordef.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM artifacts WHERE run_id = ?`, index.RunID.String()); err != nil {
		return fmt.Errorf("replace artifacts for run %s: %w", index.RunID, err)
	}
	for _, file := range index.Files {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO artifacts (id, run_id, path, size_bytes, sha256, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), index.RunID.String(), file.Path, file.SizeBytes, file.SHA256, encoding.FormatTime(time.Now()))
		if err != nil {
			return fmt.Errorf("save artifact %s for run %s: %w", file.Path, index.RunID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit artifact index for run %s: %w", index.RunID, err)
	}
	return nil
}

// GetArtifactIndex returns artifact index metadata for a run.
func (r *RunRepository) GetArtifactIndex(ctx context.Context, runID uuid.UUID) (run.ArtifactIndex, error) {
	var basePath sql.NullString
	err := r.db.GetContext(ctx, &basePath, `
		SELECT artifact_base_path
		FROM runs
		WHERE id = ?
	`, runID.String())
	if errors.Is(err, sql.ErrNoRows) {
		return run.ArtifactIndex{}, errordef.ErrNotFound
	}
	if err != nil {
		return run.ArtifactIndex{}, fmt.Errorf("get artifact base path for run %s: %w", runID, err)
	}
	if !basePath.Valid {
		return run.ArtifactIndex{}, errordef.ErrArtifactIndexMissing
	}

	var rows []record.Artifact
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT path, size_bytes, sha256
		FROM artifacts
		WHERE run_id = ?
		ORDER BY path ASC
	`, runID.String()); err != nil {
		return run.ArtifactIndex{}, fmt.Errorf("get artifacts for run %s: %w", runID, err)
	}

	index := run.NewArtifactIndex(runID, basePath.String)
	index.Files = make([]run.ArtifactFile, 0, len(rows))
	for _, row := range rows {
		index.Files = append(index.Files, row.ToArtifactFile())
	}
	return index, nil
}

func insertSpec(ctx context.Context, tx *sqlx.Tx, spec *run.Spec) error {
	row, err := record.NewSpec(spec)
	if err != nil {
		return err
	}

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO specs (
			id, project_id, name, description, model_options, data_options,
			resource_options, training_options, created_at
		)
		VALUES (
			:id, :project_id, :name, :description, :model_options, :data_options,
			:resource_options, :training_options, :created_at
		)
	`, row)
	if err != nil {
		return fmt.Errorf("insert spec %s: %w", spec.ID, err)
	}
	if err := insertSpecPresetRefs(ctx, tx, spec.ID, spec.Presets); err != nil {
		return err
	}
	return nil
}

func insertRun(ctx context.Context, tx *sqlx.Tx, rn *run.Run, projectID uuid.UUID) error {
	row := record.NewRun(rn, projectID)
	_, err := tx.NamedExecContext(ctx, `
		INSERT INTO runs (
			id, project_id, spec_id, idempotency_key, status, failure_reason,
			created_at, started_at, finished_at
		)
		VALUES (
			:id, :project_id, :spec_id, :idempotency_key, :status, :failure_reason,
			:created_at, :started_at, :finished_at
		)
	`, row)
	if err != nil {
		return fmt.Errorf("insert run %s: %w", rn.ID, err)
	}
	return nil
}

func getRunByIdempotencyKey(ctx context.Context, tx *sqlx.Tx, projectID uuid.UUID, key string) (run.Run, string, error) {
	var row record.Run
	err := tx.GetContext(ctx, &row, `
		SELECT runs.id, runs.project_id, runs.spec_id, runs.idempotency_key, runs.status,
			runs.failure_reason, runs.artifact_base_path, runs.created_at, runs.started_at,
			runs.finished_at
		FROM runs
		WHERE runs.project_id = ? AND runs.idempotency_key = ?
	`, projectID.String(), key)
	if errors.Is(err, sql.ErrNoRows) {
		return run.Run{}, "", errordef.ErrNotFound
	}
	if err != nil {
		return run.Run{}, "", fmt.Errorf("get idempotent run for project %s: %w", projectID, err)
	}
	rn, err := row.ToRun()
	if err != nil {
		return run.Run{}, "", err
	}
	spec, err := getSpecByID(ctx, tx, rn.SpecID)
	if err != nil {
		return run.Run{}, "", err
	}
	specFingerprint, err := spec.ComparableJSON()
	if err != nil {
		return run.Run{}, "", err
	}
	return rn, specFingerprint, nil
}

func getSpecByID(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (record.Spec, error) {
	var row record.Spec
	err := tx.GetContext(ctx, &row, `
		SELECT id, project_id, name, description, model_options, data_options,
			resource_options, training_options, created_at
		FROM specs
		WHERE id = ?
	`, id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return record.Spec{}, errordef.ErrNotFound
	}
	if err != nil {
		return record.Spec{}, fmt.Errorf("get spec %s: %w", id, err)
	}
	refs, err := getSpecPresetRefs(ctx, tx, id)
	if err != nil {
		return record.Spec{}, err
	}
	row.PresetRefs = refs
	return row, nil
}

func insertSpecPresetRefs(ctx context.Context, tx *sqlx.Tx, specID uuid.UUID, refs run.PresetRefs) error {
	rows := []struct {
		Category run.PresetCategory
		PresetID *run.PresetID
	}{
		{Category: run.TrainerPreset, PresetID: refs.Trainer},
		{Category: run.ResourcePreset, PresetID: refs.Resource},
		{Category: run.OutputPreset, PresetID: refs.Output},
	}
	for _, row := range rows {
		if row.PresetID == nil || *row.PresetID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO spec_preset_refs (spec_id, category, preset_id)
			VALUES (?, ?, ?)
		`, specID.String(), string(row.Category), string(*row.PresetID)); err != nil {
			return fmt.Errorf("insert spec preset ref %s %s: %w", specID, row.Category, err)
		}
	}
	return nil
}

func getSpecPresetRefs(ctx context.Context, tx *sqlx.Tx, specID uuid.UUID) (run.PresetRefs, error) {
	var rows []struct {
		Category string `db:"category"`
		PresetID string `db:"preset_id"`
	}
	if err := tx.SelectContext(ctx, &rows, `
		SELECT category, preset_id
		FROM spec_preset_refs
		WHERE spec_id = ?
	`, specID.String()); err != nil {
		return run.PresetRefs{}, fmt.Errorf("get spec preset refs %s: %w", specID, err)
	}

	var refs run.PresetRefs
	for _, row := range rows {
		presetID := run.PresetID(row.PresetID)
		switch run.PresetCategory(row.Category) {
		case run.TrainerPreset:
			refs.Trainer = &presetID
		case run.ResourcePreset:
			refs.Resource = &presetID
		case run.OutputPreset:
			refs.Output = &presetID
		}
	}
	return refs, nil
}
