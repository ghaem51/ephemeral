package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ghaem51/ephemeral/apps/control-plane/internal/domain"
	"github.com/ghaem51/ephemeral/apps/control-plane/internal/repository"
	"github.com/mattn/go-sqlite3"
)

var (
	_ repository.EnvironmentRepository = (*EnvironmentRepository)(nil)
	_ repository.WorkflowRepository    = (*WorkflowRepository)(nil)
)

type Store struct {
	db *sql.DB
}

type EnvironmentRepository struct {
	db *sql.DB
}

type WorkflowRepository struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// SQLite foreign-key settings are connection-local. A single connection also
	// avoids surprising behavior with in-memory databases in repository tests.
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.initialize(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Environments() *EnvironmentRepository {
	return &EnvironmentRepository{db: s.db}
}

func (s *Store) Workflows() *WorkflowRepository {
	return &WorkflowRepository{db: s.db}
}

func mapWriteError(err error) error {
	var sqliteError sqlite3.Error
	if errors.As(err, &sqliteError) && sqliteError.ExtendedCode == sqlite3.ErrConstraintUnique {
		return fmt.Errorf("%w: %v", domain.ErrAlreadyExists, err)
	}
	return err
}

type scanner interface {
	Scan(...any) error
}
