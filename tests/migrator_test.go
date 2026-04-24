package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinuthankachan/migrator/internal/config"
	"github.com/jinuthankachan/migrator/pkg/migrator"
)

func TestInit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").WillReturnResult(sqlmock.NewResult(0, 0))

	if err := migrator.Init(db); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestUpAllPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	migrationsDir := filepath.Join(dir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	os.WriteFile(filepath.Join(migrationsDir, "001.sql"), []byte("CREATE TABLE users (id INT);"), 0644)
	os.WriteFile(filepath.Join(migrationsDir, "002.sql"), []byte("CREATE INDEX idx ON users(id);"), 0644)

	configContent := `directory: migrations
files:
  - 001.sql
  - 002.sql
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	mock.ExpectBegin()

	rows1 := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("001.sql").WillReturnRows(rows1)
	mock.ExpectExec("CREATE TABLE users").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations").WithArgs("001.sql", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	rows2 := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("002.sql").WillReturnRows(rows2)
	mock.ExpectExec("CREATE INDEX idx").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations").WithArgs("002.sql", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	if err := migrator.Up(db, configPath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestUpSkipsExecuted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	migrationsDir := filepath.Join(dir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	os.WriteFile(filepath.Join(migrationsDir, "001.sql"), []byte("SELECT 1;"), 0644)
	os.WriteFile(filepath.Join(migrationsDir, "002.sql"), []byte("SELECT 2;"), 0644)

	configContent := `directory: migrations
files:
  - 001.sql
  - 002.sql
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	mock.ExpectBegin()

	rows1 := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("001.sql").WillReturnRows(rows1)

	rows2 := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("002.sql").WillReturnRows(rows2)
	mock.ExpectExec("SELECT 2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations").WithArgs("002.sql", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	if err := migrator.Up(db, configPath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestUpRollbackOnFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	migrationsDir := filepath.Join(dir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	os.WriteFile(filepath.Join(migrationsDir, "001.sql"), []byte("INVALID SQL;"), 0644)

	configContent := `directory: migrations
files:
  - 001.sql
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	mock.ExpectBegin()

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("001.sql").WillReturnRows(rows)
	mock.ExpectExec("INVALID SQL").WillReturnError(fmt.Errorf("syntax error"))
	mock.ExpectRollback()

	if err := migrator.Up(db, configPath); err == nil {
		t.Fatal("expected error for invalid SQL")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestUpFile(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "001.sql")
	os.WriteFile(filePath, []byte("SELECT 1;"), 0644)

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("001.sql").WillReturnRows(rows)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations").WithArgs("001.sql", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := migrator.UpFile(db, filePath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestUpFileSkipsExecuted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "001.sql")
	os.WriteFile(filePath, []byte("SELECT 1;"), 0644)

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("001.sql").WillReturnRows(rows)

	if err := migrator.UpFile(db, filePath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestAdd(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configContent := `directory: migrations
files:
  - 001.sql
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	if err := migrator.Add(configPath, "002_new"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	filePath := filepath.Join(dir, "migrations", "002_new.sql")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", filePath)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Files) != 2 || cfg.Files[1] != "002_new.sql" {
		t.Errorf("expected config to contain 002_new.sql, got %v", cfg.Files)
	}
}

func TestAddDuplicate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configContent := `directory: migrations
files:
  - 001.sql
`
	os.WriteFile(configPath, []byte(configContent), 0644)
	os.MkdirAll(filepath.Join(dir, "migrations"), 0755)
	os.WriteFile(filepath.Join(dir, "migrations", "001.sql"), []byte(""), 0644)

	err := migrator.Add(configPath, "001")
	if err == nil {
		t.Fatal("expected error for duplicate file")
	}
}

func TestAddExtensionAppended(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configContent := `directory: migrations
files:
  - 001.sql
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	if err := migrator.Add(configPath, "003_test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	filePath := filepath.Join(dir, "migrations", "003_test.sql")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", filePath)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	found := false
	for _, f := range cfg.Files {
		if f == "003_test.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected config to contain 003_test.sql, got %v", cfg.Files)
	}
}
