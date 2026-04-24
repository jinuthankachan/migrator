package migrator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jinuthankachan/migrator/internal/config"
	_ "github.com/lib/pq"
)

// Init creates the migrations tracking table if it does not exist.
func Init(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			filename VARCHAR(255) NOT NULL UNIQUE,
			executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

// isExecuted checks whether a migration file has already been run.
func isExecuted(db *sql.DB, filename string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM migrations WHERE filename = $1)`
	err := db.QueryRow(query, filename).Scan(&exists)
	return exists, err
}

// recordMigration inserts a migration record after successful execution.
func recordMigration(tx *sql.Tx, filename string) error {
	query := `INSERT INTO migrations (filename, executed_at) VALUES ($1, $2)`
	_, err := tx.Exec(query, filename, time.Now())
	return err
}

// executeFile runs a single SQL migration file within a transaction.
func executeFile(db *sql.DB, filePath string) error {
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for %s: %w", filePath, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", filePath, err)
	}

	filename := filepath.Base(filePath)
	if err := recordMigration(tx, filename); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", filename, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", filename, err)
	}

	return nil
}

// Up runs all pending migrations defined in the configuration file.
// All pending migrations are executed within a single database transaction,
// ensuring atomicity: either all succeed or all are rolled back.
func Up(db *sql.DB, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, file := range cfg.Files {
		executed, err := isExecuted(db, file)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", file, err)
		}
		if executed {
			continue
		}

		filePath := filepath.Join(filepath.Dir(configPath), cfg.Directory, file)
		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
		}

		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filePath, err)
		}

		if err := recordMigration(tx, file); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	return nil
}

// UpFile runs a single migration file.
func UpFile(db *sql.DB, filePath string) error {
	filename := filepath.Base(filePath)
	executed, err := isExecuted(db, filename)
	if err != nil {
		return fmt.Errorf("failed to check migration status for %s: %w", filename, err)
	}
	if executed {
		return nil
	}

	return executeFile(db, filePath)
}

// Add creates a new empty migration file and appends it to the config.
func Add(configPath, filename string) error {
	if !strings.HasSuffix(filename, ".sql") {
		filename += ".sql"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dir := filepath.Join(filepath.Dir(configPath), cfg.Directory)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	filePath := filepath.Join(dir, filename)
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("migration file already exists: %s", filename)
	}

	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	cfg.Files = append(cfg.Files, filename)
	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}
