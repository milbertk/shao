package dbmanager

import (
	"database/sql"
	"fmt"
	"log"
)

// Executor is what both *sql.DB and *sql.Tx can do. Data functions that may
// run inside a transaction take an Executor (2026-10-24).
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// WithTx runs fn inside ONE database transaction: everything is saved
// (COMMIT) or nothing is (ROLLBACK) when fn returns an error or panics.
// Multi-step saves use it so they never stay half-saved.
func WithTx(fn func(tx *sql.Tx) error) (err error) {
	db, err := GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("[dbmanager] rollback failed: %v", rollbackErr)
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
