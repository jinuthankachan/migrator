package main

import (
	"fmt"

	"github.com/jinuthankachan/migrator/pkg/migrator"
	"github.com/spf13/cobra"
)

var (
	upFile   string
	upConfig string
)

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&upFile, "file", "f", "", "Path to a single migration file to execute")
	upCmd.Flags().StringVarP(&upConfig, "config", "c", "", "Path to the configuration file")
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Run pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		if upFile != "" {
			if err := migrator.UpFile(db, upFile); err != nil {
				return fmt.Errorf("failed to run migration file: %w", err)
			}
			fmt.Println("Migration executed successfully.")
			return nil
		}

		if upConfig == "" {
			return fmt.Errorf("either --file or --config must be specified")
		}

		if err := migrator.Up(db, upConfig); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		fmt.Println("Migrations executed successfully.")
		return nil
	},
}
