package sqlite

import (
	"context"
	"fmt"
	"time"
)

const staleWorkflowMessage = "workflow interrupted by an unclean control-plane shutdown; retry or destroy the environment"

type RecoveredWorkflow struct {
	WorkflowID    string
	EnvironmentID string
}

// RecoverStaleWorkflows atomically converts workflows that cannot still be
// executing after process startup into auditable failures.
func (s *Store) RecoverStaleWorkflows(ctx context.Context, at time.Time) ([]RecoveredWorkflow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin stale workflow recovery: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, environment_id FROM workflows WHERE status = 'RUNNING' ORDER BY rowid`)
	if err != nil {
		return nil, fmt.Errorf("list stale workflows: %w", err)
	}
	var recovered []RecoveredWorkflow
	for rows.Next() {
		var item RecoveredWorkflow
		if err := rows.Scan(&item.WorkflowID, &item.EnvironmentID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan stale workflow: %w", err)
		}
		recovered = append(recovered, item)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close stale workflow rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale workflows: %w", err)
	}

	completedAt := formatTime(at)
	for _, item := range recovered {
		if _, err := tx.ExecContext(ctx, `
			UPDATE workflow_steps
			SET status = 'FAILED', message = ?, error_message = ?, completed_at = ?
			WHERE workflow_id = ? AND status = 'RUNNING'`,
			staleWorkflowMessage, staleWorkflowMessage, completedAt, item.WorkflowID,
		); err != nil {
			return nil, fmt.Errorf("fail running step for workflow %q: %w", item.WorkflowID, err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE workflows SET status = 'FAILED', completed_at = ? WHERE id = ? AND status = 'RUNNING'`,
			completedAt, item.WorkflowID,
		); err != nil {
			return nil, fmt.Errorf("fail stale workflow %q: %w", item.WorkflowID, err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'FAILED', error_message = ?, updated_at = ?
			WHERE id = ? AND status IN ('PENDING', 'PROVISIONING', 'DESTROYING')`,
			staleWorkflowMessage, completedAt, item.EnvironmentID,
		); err != nil {
			return nil, fmt.Errorf("make environment %q recoverable: %w", item.EnvironmentID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit stale workflow recovery: %w", err)
	}
	return recovered, nil
}
