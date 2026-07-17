package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

const workflowColumns = `id, environment_id, operation, status, started_at, completed_at`
const stepColumns = `id, workflow_id, name, step_order, status, message, error_message, started_at, completed_at`

func (r *WorkflowRepository) CreateWithSteps(ctx context.Context, workflow *domain.Workflow) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create workflow: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
        INSERT INTO workflows (`+workflowColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
		workflow.ID, workflow.EnvironmentID, workflow.Operation, workflow.Status,
		formatOptionalTime(workflow.StartedAt), formatOptionalTime(workflow.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("create workflow: %w", mapWriteError(err))
	}

	for i := range workflow.Steps {
		step := &workflow.Steps[i]
		if step.WorkflowID == "" {
			step.WorkflowID = workflow.ID
		}
		if step.WorkflowID != workflow.ID {
			return fmt.Errorf("create workflow step %q: %w: workflow ID does not match", step.ID, domain.ErrValidation)
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO workflow_steps (`+stepColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			step.ID, step.WorkflowID, step.Name, step.Order, step.Status, step.Message,
			step.ErrorMessage, formatOptionalTime(step.StartedAt), formatOptionalTime(step.CompletedAt),
		); err != nil {
			return fmt.Errorf("create workflow step %q: %w", step.ID, mapWriteError(err))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create workflow: %w", err)
	}
	return nil
}

func (r *WorkflowRepository) Update(ctx context.Context, workflow *domain.Workflow) error {
	result, err := r.db.ExecContext(ctx, `
        UPDATE workflows SET operation = ?, status = ?, started_at = ?, completed_at = ? WHERE id = ?`,
		workflow.Operation, workflow.Status, formatOptionalTime(workflow.StartedAt),
		formatOptionalTime(workflow.CompletedAt), workflow.ID,
	)
	if err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}
	return requireUpdated(result, "workflow", workflow.ID)
}

func (r *WorkflowRepository) UpdateStep(ctx context.Context, step *domain.WorkflowStep) error {
	result, err := r.db.ExecContext(ctx, `
        UPDATE workflow_steps
        SET name = ?, step_order = ?, status = ?, message = ?, error_message = ?, started_at = ?, completed_at = ?
        WHERE id = ? AND workflow_id = ?`,
		step.Name, step.Order, step.Status, step.Message, step.ErrorMessage,
		formatOptionalTime(step.StartedAt), formatOptionalTime(step.CompletedAt), step.ID, step.WorkflowID,
	)
	if err != nil {
		return fmt.Errorf("update workflow step: %w", mapWriteError(err))
	}
	return requireUpdated(result, "workflow step", step.ID)
}

func (r *WorkflowRepository) GetLatestForEnvironment(ctx context.Context, environmentID string) (*domain.Workflow, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
        SELECT id FROM workflows WHERE environment_id = ? ORDER BY rowid DESC LIMIT 1`, environmentID).Scan(&id)
	if err != nil {
		return nil, mapReadError(err, "workflow for environment", environmentID)
	}
	return r.GetWithSteps(ctx, id)
}

func (r *WorkflowRepository) GetWithSteps(ctx context.Context, id string) (*domain.Workflow, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin get workflow: %w", err)
	}
	defer tx.Rollback()

	workflow, err := scanWorkflow(tx.QueryRowContext(ctx,
		`SELECT `+workflowColumns+` FROM workflows WHERE id = ?`, id))
	if err != nil {
		return nil, mapReadError(err, "workflow", id)
	}

	rows, err := tx.QueryContext(ctx, `
        SELECT `+stepColumns+` FROM workflow_steps WHERE workflow_id = ? ORDER BY step_order`, id)
	if err != nil {
		return nil, fmt.Errorf("get workflow steps: %w", err)
	}
	defer rows.Close()

	workflow.Steps = make([]domain.WorkflowStep, 0)
	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return nil, fmt.Errorf("scan workflow step: %w", err)
		}
		workflow.Steps = append(workflow.Steps, *step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow steps: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit get workflow: %w", err)
	}
	return workflow, nil
}

func scanWorkflow(row scanner) (*domain.Workflow, error) {
	var workflow domain.Workflow
	var startedAt, completedAt sql.NullString
	if err := row.Scan(&workflow.ID, &workflow.EnvironmentID, &workflow.Operation, &workflow.Status, &startedAt, &completedAt); err != nil {
		return nil, err
	}
	var err error
	workflow.StartedAt, err = parseOptionalTime(startedAt)
	if err != nil {
		return nil, err
	}
	workflow.CompletedAt, err = parseOptionalTime(completedAt)
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

func scanStep(row scanner) (*domain.WorkflowStep, error) {
	var step domain.WorkflowStep
	var startedAt, completedAt sql.NullString
	if err := row.Scan(
		&step.ID, &step.WorkflowID, &step.Name, &step.Order, &step.Status,
		&step.Message, &step.ErrorMessage, &startedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	var err error
	step.StartedAt, err = parseOptionalTime(startedAt)
	if err != nil {
		return nil, err
	}
	step.CompletedAt, err = parseOptionalTime(completedAt)
	if err != nil {
		return nil, err
	}
	return &step, nil
}
