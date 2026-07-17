package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
)

const environmentColumns = `id, name, image, container_port, host_port, container_id, url, status, error_message, created_at, updated_at`

func (r *EnvironmentRepository) Create(ctx context.Context, environment *domain.Environment) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO environments (`+environmentColumns+`)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		environment.ID, environment.Name, environment.Image, environment.ContainerPort,
		environment.HostPort, environment.ContainerID, environment.URL, environment.Status,
		environment.ErrorMessage, formatTime(environment.CreatedAt), formatTime(environment.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create environment: %w", mapWriteError(err))
	}
	return nil
}

func (r *EnvironmentRepository) Update(ctx context.Context, environment *domain.Environment) error {
	result, err := r.db.ExecContext(ctx, `
        UPDATE environments
        SET name = ?, image = ?, container_port = ?, host_port = ?, container_id = ?,
            url = ?, status = ?, error_message = ?, created_at = ?, updated_at = ?
        WHERE id = ?`,
		environment.Name, environment.Image, environment.ContainerPort, environment.HostPort,
		environment.ContainerID, environment.URL, environment.Status, environment.ErrorMessage,
		formatTime(environment.CreatedAt), formatTime(environment.UpdatedAt), environment.ID,
	)
	if err != nil {
		return fmt.Errorf("update environment: %w", mapWriteError(err))
	}
	return requireUpdated(result, "environment", environment.ID)
}

func (r *EnvironmentRepository) GetByID(ctx context.Context, id string) (*domain.Environment, error) {
	environment, err := scanEnvironment(r.db.QueryRowContext(ctx,
		`SELECT `+environmentColumns+` FROM environments WHERE id = ?`, id))
	return environment, mapReadError(err, "environment", id)
}

func (r *EnvironmentRepository) FindByName(ctx context.Context, name string) (*domain.Environment, error) {
	environment, err := scanEnvironment(r.db.QueryRowContext(ctx,
		`SELECT `+environmentColumns+` FROM environments WHERE name = ?
         ORDER BY CASE WHEN status = 'DESTROYED' THEN 1 ELSE 0 END, created_at DESC LIMIT 1`, name))
	return environment, mapReadError(err, "environment", name)
}

func (r *EnvironmentRepository) List(ctx context.Context) ([]domain.Environment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+environmentColumns+` FROM environments ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	environments := make([]domain.Environment, 0)
	for rows.Next() {
		environment, err := scanEnvironment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		environments = append(environments, *environment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate environments: %w", err)
	}
	return environments, nil
}

func scanEnvironment(row scanner) (*domain.Environment, error) {
	var environment domain.Environment
	var createdAt, updatedAt string
	if err := row.Scan(
		&environment.ID, &environment.Name, &environment.Image, &environment.ContainerPort,
		&environment.HostPort, &environment.ContainerID, &environment.URL, &environment.Status,
		&environment.ErrorMessage, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}

	var err error
	environment.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	environment.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, err
	}
	return &environment, nil
}

func mapReadError(err error, entity, identifier string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s %q: %w", entity, identifier, domain.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("get %s %q: %w", entity, identifier, err)
	}
	return nil
}

func requireUpdated(result sql.Result, entity, identifier string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected %s rows: %w", entity, err)
	}
	if count == 0 {
		return fmt.Errorf("%s %q: %w", entity, identifier, domain.ErrNotFound)
	}
	return nil
}
