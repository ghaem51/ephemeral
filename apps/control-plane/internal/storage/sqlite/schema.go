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
    application_version TEXT NOT NULL DEFAULT '',
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
	if err := s.ensureApplicationVersionColumn(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureApplicationVersionColumn(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info(environments)")
	if err != nil {
		return fmt.Errorf("inspect environments schema: %w", err)
	}
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return fmt.Errorf("scan environments schema: %w", err)
		}
		if name == "application_version" {
			found = true
		}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close environments schema rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate environments schema: %w", err)
	}
	if found {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE environments ADD COLUMN application_version TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add environment application version column: %w", err)
	}
	return nil
}
