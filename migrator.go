package migrator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// GetDB is a global function variable to allow overriding the database connection for testing.
var GetDB = Connect

// Init creates the migration table in the database if it doesn't exist.
func Init() error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
	CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		file_name VARCHAR(255) UNIQUE NOT NULL,
		executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// Up runs the migrations specified in the configuration file.
func Up(configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := GetDB()
	if err != nil {
		return err
	}
	defer db.Close()

	for _, file := range cfg.Files {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)", file).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", file, err)
		}

		if exists {
			log.Printf("Skipping %s, already executed", file)
			continue
		}

		filePath := filepath.Join(cfg.Directory, file)
		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		if _, err := tx.Exec("INSERT INTO migrations (file_name, executed_at) VALUES ($1, $2)", file, time.Now()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", file, err)
		}

		log.Printf("Successfully executed %s", file)
	}

	return nil
}

// Add creates a new migration file and adds it to the configuration.
func Add(filename, configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if filepath.Ext(filename) != ".sql" {
		filename += ".sql"
	}

	if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory %s: %w", cfg.Directory, err)
	}

	filePath := filepath.Join(cfg.Directory, filename)
	
	content := fmt.Sprintf("-- Migration: %s\n", filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create migration file %s: %w", filePath, err)
	}

	cfg.Files = append(cfg.Files, filename)
	if err := SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
