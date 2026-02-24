package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jinuthankachan/migrator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrator <command> [arguments]")
		fmt.Println("Commands:")
		fmt.Println("  init    Initialize the database")
		fmt.Println("  up      Run migrations")
		fmt.Println("  add     Add a new migration file")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "init":
		if err := migrator.Init(); err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		fmt.Println("Initialized migrations table successfully.")

	case "up":
		upCmd := flag.NewFlagSet("up", flag.ExitOnError)
		configPath := upCmd.String("c", "", "Path to the config file")
		upCmd.Parse(os.Args[2:])

		if *configPath == "" {
			log.Fatal("Config file path is required (-c)")
		}

		if err := migrator.Up(*configPath); err != nil {
			log.Fatalf("Up failed: %v", err)
		}
		fmt.Println("Migrations completed successfully.")

	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		filename := addCmd.String("f", "", "Filename for the new migration")
		configPath := addCmd.String("c", "", "Path to the config file")
		addCmd.Parse(os.Args[2:])

		if *filename == "" || *configPath == "" {
			log.Fatal("Filename (-f) and Config file path (-c) are required")
		}

		if err := migrator.Add(*filename, *configPath); err != nil {
			log.Fatalf("Add failed: %v", err)
		}
		fmt.Println("Migration file added successfully.")

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
