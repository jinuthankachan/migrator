package migrator

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func mockDB() (*sql.DB, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		return nil, nil, err
	}
	return db, mock, nil
}

func TestConnect(t *testing.T) {
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "pass")
	os.Setenv("DB_NAME", "db")
	defer os.Clearenv()

	db, err := Connect()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if db == nil {
		t.Fatalf("expected db to be non-nil")
	}
	db.Close()
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	os.WriteFile(configPath, []byte("invalid yaml"), 0644)

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Errorf("expected error for invalid config")
	}
}

func TestSaveConfig_Error(t *testing.T) {
	err := SaveConfig("/invalid/path/config.json", &Config{})
	if err == nil {
		t.Errorf("expected error when saving to invalid path")
	}
}

func TestInit_DBError(t *testing.T) {
	GetDB = func() (*sql.DB, error) { return nil, errors.New("db connection failed") }
	defer func() { GetDB = Connect }()

	err := Init()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestInit_ExecError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	mock.ExpectExec(`
	CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		file_name VARCHAR(255) UNIQUE NOT NULL,
		executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`).WillReturnError(errors.New("exec error"))

	err = Init()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestInit(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	mock.ExpectExec(`
	CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		file_name VARCHAR(255) UNIQUE NOT NULL,
		executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`).WillReturnResult(sqlmock.NewResult(1, 1))

	err = Init()
	if err != nil {
		t.Errorf("error was not expected while updating stats: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUp_LoadConfigError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	os.WriteFile(configPath, []byte("invalid yaml"), 0644)

	err := Up(configPath)
	if err == nil {
		t.Errorf("expected error for invalid config")
	}
}

func TestUp_DBConnectError(t *testing.T) {
	GetDB = func() (*sql.DB, error) { return nil, errors.New("db error") }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{"test.sql"}})

	err := Up(configPath)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestUp_QueryError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{"test.sql"}})

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").WillReturnError(errors.New("query error"))

	err = Up(configPath)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestUp_FileReadError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{"test.sql"}})

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err = Up(configPath)
	if err == nil {
		t.Errorf("expected error for missing file")
	}
}

func TestUp_TransactionError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	migrationFile := "test_migration.sql"
	os.WriteFile(filepath.Join(tempDir, migrationFile), []byte("SQL"), 0644)
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{migrationFile}})

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin().WillReturnError(errors.New("tx error"))

	err = Up(configPath)
	if err == nil {
		t.Errorf("expected error for transaction begin")
	}
}

func TestUp_ExecError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	migrationFile := "test_migration.sql"
	os.WriteFile(filepath.Join(tempDir, migrationFile), []byte("SQL"), 0644)
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{migrationFile}})

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec("SQL").WillReturnError(errors.New("exec error"))
	mock.ExpectRollback()

	err = Up(configPath)
	if err == nil {
		t.Errorf("expected error for transaction exec")
	}
}

func TestUp_InsertError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	migrationFile := "test_migration.sql"
	os.WriteFile(filepath.Join(tempDir, migrationFile), []byte("SQL"), 0644)
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{migrationFile}})

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec("SQL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations (file_name, executed_at) VALUES ($1, $2)").WillReturnError(errors.New("insert error"))
	mock.ExpectRollback()

	err = Up(configPath)
	if err == nil {
		t.Errorf("expected error for transaction insert")
	}
}

func TestUp_CommitError(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	migrationFile := "test_migration.sql"
	os.WriteFile(filepath.Join(tempDir, migrationFile), []byte("SQL"), 0644)
	SaveConfig(configPath, &Config{Directory: tempDir, Files: []string{migrationFile}})

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec("SQL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations (file_name, executed_at) VALUES ($1, $2)").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err = Up(configPath)
	if err == nil {
		t.Errorf("expected error for transaction commit")
	}
}

func TestUp(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error setting up mock db: %s", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	migrationFile := "test_migration.sql"
	migrationContent := "CREATE TABLE test_table (id INT);"

	err = os.WriteFile(filepath.Join(tempDir, migrationFile), []byte(migrationContent), 0644)
	if err != nil {
		t.Fatalf("failed to create migration file: %v", err)
	}

	cfg := &Config{
		Directory: tempDir,
		Files:     []string{migrationFile},
	}
	err = SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").
		WithArgs(migrationFile).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectBegin()
	mock.ExpectExec(migrationContent).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO migrations (file_name, executed_at) VALUES ($1, $2)").
		WithArgs(migrationFile, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := Up(configPath); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpAlreadyExecuted(t *testing.T) {
	db, mock, err := mockDB()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	GetDB = func() (*sql.DB, error) { return db, nil }
	defer func() { GetDB = Connect }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	migrationFile := "executed_migration.sql"

	cfg := &Config{Directory: tempDir, Files: []string{migrationFile}}
	err = SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	mock.ExpectQuery("SELECT EXISTS(SELECT 1 FROM migrations WHERE file_name = $1)").
		WithArgs(migrationFile).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	if err := Up(configPath); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestAdd_LoadConfigError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	os.WriteFile(configPath, []byte("invalid yaml"), 0644)

	err := Add("new_migration", configPath)
	if err == nil {
		t.Errorf("expected error for invalid config")
	}
}

func TestAdd_WriteError(t *testing.T) {
	err := Add("new_migration", "/invalid/path/config.json")
	if err == nil {
		t.Errorf("expected error for invalid directory")
	}
}

func TestAdd(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	cfgDir := filepath.Join(tempDir, "migrations")
	cfg := &Config{Directory: cfgDir, Files: []string{}}
	SaveConfig(configPath, cfg)

	err := Add("new_migration", configPath)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(loadedCfg.Files) != 1 || loadedCfg.Files[0] != "new_migration.sql" {
		t.Errorf("expected config to have new_migration.sql, got %v", loadedCfg.Files)
	}

	_, err = os.Stat(filepath.Join(cfgDir, "new_migration.sql"))
	if err != nil {
		t.Errorf("expected migration file to be created, got error: %v", err)
	}
}
