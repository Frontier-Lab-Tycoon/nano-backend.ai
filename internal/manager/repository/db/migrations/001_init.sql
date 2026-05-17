CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS specs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    model_options TEXT NOT NULL,
    data_options TEXT NOT NULL,
    resource_options TEXT NOT NULL,
    training_options TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS preset_categories (
    id TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

INSERT OR IGNORE INTO preset_categories (id, description) VALUES
    ('trainer', 'Trainer runtime and training parameter presets'),
    ('resource', 'Resource default and policy presets'),
    ('output', 'Output and artifact policy presets');

CREATE TABLE IF NOT EXISTS presets (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL REFERENCES preset_categories(id),
    display_name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS trainer_presets (
    preset_id TEXT PRIMARY KEY REFERENCES presets(id) ON DELETE CASCADE,
    image TEXT NOT NULL,
    entrypoint TEXT NOT NULL,
    env TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS preset_option_rules (
    preset_id TEXT NOT NULL REFERENCES presets(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value_type TEXT NOT NULL,
    min_value REAL,
    max_value REAL,
    PRIMARY KEY(preset_id, key)
);

CREATE TABLE IF NOT EXISTS preset_default_values (
    preset_id TEXT NOT NULL REFERENCES presets(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value_json TEXT NOT NULL,
    PRIMARY KEY(preset_id, key)
);

INSERT OR IGNORE INTO presets (id, category, display_name, enabled, created_at) VALUES
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'trainer', 'Axolotl LoRA SFT', 1, '1970-01-01T00:00:00Z'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'trainer', 'Unsloth LoRA SFT', 1, '1970-01-01T00:00:00Z');

INSERT OR IGNORE INTO trainer_presets (preset_id, image, entrypoint, env) VALUES
    (
        '16f6f42a-597b-4c37-9b8e-7f3908fbfa73',
        'axolotl:latest',
        '["axolotl","train","/workspace/resolved_config.yaml"]',
        '{"HF_HOME":"/cache/huggingface"}'
    ),
    (
        '258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75',
        'unsloth:latest',
        '["python","-m","nano_backend.train_unsloth","--config","/workspace/resolved_config.yaml"]',
        '{"HF_HOME":"/cache/huggingface"}'
    );

INSERT OR IGNORE INTO preset_option_rules (preset_id, key, value_type, min_value, max_value) VALUES
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'learning_rate', 'float', 0, 1),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'num_epochs', 'int', 1, 100),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'max_seq_length', 'int', 128, 32768),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'lora_r', 'int', 1, 256),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'lora_alpha', 'int', 1, 512),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'micro_batch_size', 'int', 1, 64),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'learning_rate', 'float', 0, 1),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'num_epochs', 'int', 1, 100),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'max_seq_length', 'int', 128, 32768),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'lora_r', 'int', 1, 256),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'lora_alpha', 'int', 1, 512),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'micro_batch_size', 'int', 1, 64);

INSERT OR IGNORE INTO preset_default_values (preset_id, key, value_json) VALUES
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'learning_rate', '0.0002'),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'num_epochs', '3'),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'max_seq_length', '4096'),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'lora_r', '16'),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'lora_alpha', '32'),
    ('16f6f42a-597b-4c37-9b8e-7f3908fbfa73', 'micro_batch_size', '1'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'learning_rate', '0.0002'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'num_epochs', '3'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'max_seq_length', '4096'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'lora_r', '16'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'lora_alpha', '32'),
    ('258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75', 'micro_batch_size', '1');

CREATE TABLE IF NOT EXISTS spec_preset_refs (
    spec_id TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
    category TEXT NOT NULL REFERENCES preset_categories(id),
    preset_id TEXT NOT NULL REFERENCES presets(id),
    PRIMARY KEY(spec_id, category)
);

CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    spec_id TEXT NOT NULL REFERENCES specs(id),
    idempotency_key TEXT,
    status TEXT NOT NULL,
    failure_reason TEXT,
    artifact_base_path TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    UNIQUE(project_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_runs_project_created_at
    ON runs(project_id, created_at);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(run_id, path)
);
