# Migrator
A simple and lightweight tool for PostgreSQL database migration operations written in Go.
This can be used seamlessly as a standalone CLI binary or integrated directly into your Go projects as a library package.

## Features
- **Transactional Safety:** Each migration file executes within its own database transaction. If a statement fails, the entire migration rolls back safely without leaving partial updates.
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
* Execute all pending migrations sequentially:
```bash
migrator up -c ./config.yaml
```
* Execute a single file 
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

### 2. Implementation Example
```go
package main

import (
	"log"
	"github.com/jinuthankachan/migrator"
)

func main() {
	// Initialize the tracking table (safe to run on every startup)
	dbConn, err := db.Connect()
	if err := migrator.Init(dbConn); err != nil {
		log.Fatalf("Failed to initialize migrations table: %v", err)
	}

	// Run all pending migrations
	if err := migrator.Up("path/to/your/config.yaml"); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Database is up-to-date!")
}
```
