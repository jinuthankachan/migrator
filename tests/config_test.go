package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jinuthankachan/migrator/internal/config"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `directory: migrations
files:
  - 001_create_users.sql
  - 002_add_index.sql
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Directory != "migrations" {
		t.Errorf("expected directory 'migrations', got %s", cfg.Directory)
	}
	if len(cfg.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(cfg.Files))
	}
	if cfg.Files[0] != "001_create_users.sql" {
		t.Errorf("expected first file '001_create_users.sql', got %s", cfg.Files[0])
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := config.Load("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &config.Config{
		Directory: "migrations",
		Files:     []string{"001.sql", "002.sql"},
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if loaded.Directory != cfg.Directory {
		t.Errorf("expected directory %s, got %s", cfg.Directory, loaded.Directory)
	}
	if len(loaded.Files) != len(cfg.Files) {
		t.Errorf("expected %d files, got %d", len(cfg.Files), len(loaded.Files))
	}
	for i, f := range cfg.Files {
		if loaded.Files[i] != f {
			t.Errorf("expected file %s, got %s", f, loaded.Files[i])
		}
	}
}
