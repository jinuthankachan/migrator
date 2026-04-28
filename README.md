# Migrator
A simple and lightweight tool for PostgreSQL database migration operations written in Go.
This can be used seamlessly as a standalone CLI binary or integrated directly into your Go projects as a library package.

## Features
- **Transactional Safety:** When running migrations via `migrator up`, all pending migrations execute within a single database transaction. If any statement fails, the entire batch rolls back atomically. Single-file execution (`migrator up -f`) also runs within its own transaction.
- **Sequential Execution:** Migrations are executed strictly in the order specified in a simple configuration file.
- **Idempotency:** A `migrations` tracking table ensures that already-executed migrations are skipped.

## Requirements
- Go 1.25.0 or later
- PostgreSQL database

## Installation & Configuration

### The Config File (`config.yaml`)
Migrator uses a YAML configuration file to track where your migration files are stored and the order in which they should be executed. See [`config.example.yaml`](config.example.yaml) for a complete example.

**Example `config.yaml`:**
```yaml
directory: migrations
files:
  - 001_create_users_table.sql
  - 002_add_email_index.sql
```
*Tip: When you use the `migrator add` command, this file is automatically updated.*

---

## Standalone Binary Usage

### Environment Variables
Migrator in standalone mode relies on environment variables to connect to your PostgreSQL instance. Set these before running any commands:

* `DB_HOST` - The host address of the database (e.g., `localhost`).
* `DB_PORT` - The port number of the database (e.g., `5432`).
* `DB_USER` - The username for the database.
* `DB_PASSWORD` - The password for the database.
* `DB_NAME` - The name of the database.

### 1. Installation
Install the CLI tool globally via `go install`:
```bash
go install github.com/jinuthankachan/migrator/cmd/migrator@latest
```

### 2. Initialization
Create the internal `migrations` tracking table in your database. You only need to run this once per database setup.
```bash
migrator init
```

### 3. Adding Migrations
Create a new migration file. This will generate a `.sql` file with the specified name in the configured directory and automatically add the filename to your `config.yaml`.
```bash
migrator add -f 001_create_users_table.sql -c ./config.yaml
```
*(Note: The `.sql` extension is automatically appended if omitted).*

### 4. Running Migrations
* Execute all pending migrations sequentially (atomic batch):
```bash
migrator up -c ./config.yaml
```
* Execute a single file:
```bash
migrator up -f ./migrations/001_create_users_table.sql
```

---

## Library Package Usage

You can embed Migrator directly into your Go application to run migrations automatically on startup or via your own subcommands.

### 1. Import
Add the package to your project:
```bash
go get github.com/jinuthankachan/migrator
```

### 2. Library API Reference

| Function | Description |
|----------|-------------|
| `migrator.Init(db *sql.DB) error` | Creates the `migrations` tracking table if it doesn't exist. Safe to call on every startup. |
| `migrator.Up(db *sql.DB, configPath string) error` | Runs all pending migrations from the config in a single atomic transaction. |
| `migrator.UpFile(db *sql.DB, filePath string) error` | Runs a single migration file in its own transaction. |
| `migrator.Add(configPath, filename string) error` | Creates a new empty `.sql` file and appends it to `config.yaml`. |

### 3. Implementation Example
```go
package main

import (
	"database/sql"
	"log"

	"github.com/jinuthankachan/migrator"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost user=dbuser password=dbpass dbname=mydb sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize the tracking table (safe to run on every startup)
	if err := migrator.Init(db); err != nil {
		log.Fatalf("Failed to initialize migrations table: %v", err)
	}

	// Run all pending migrations atomically
	if err := migrator.Up(db, "path/to/your/config.yaml"); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Database is up-to-date!")
}
```
