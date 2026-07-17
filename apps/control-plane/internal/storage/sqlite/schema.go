package sqlite

import (
	"context"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS environments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    container_port INTEGER NOT NULL,
    host_port INTEGER NOT NULL,
    container_id TEXT NOT NULL,
    url TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_environments_active_name
    ON environments(name) WHERE status != 'DESTROYED';

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    environment_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workflows_environment
    ON workflows(environment_id);

CREATE TABLE IF NOT EXISTS workflow_steps (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    name TEXT NOT NULL,
    step_order INTEGER NOT NULL,
    status TEXT NOT NULL,
    message TEXT NOT NULL,
    error_message TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    UNIQUE (workflow_id, step_order)
);

CREATE INDEX IF NOT EXISTS idx_workflow_steps_workflow
    ON workflow_steps(workflow_id, step_order);
`

func (s *Store) initialize(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize sqlite schema: %w", err)
	}
	return nil
}
