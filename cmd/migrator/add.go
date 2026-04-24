package main

import (
	"fmt"

	"github.com/jinuthankachan/migrator/pkg/migrator"
	"github.com/spf13/cobra"
)

var (
	addFileName string
	addConfig   string
)

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addFileName, "file", "f", "", "Name of the migration file to create")
	addCmd.Flags().StringVarP(&addConfig, "config", "c", "config.yaml", "Path to the configuration file")
	_ = addCmd.MarkFlagRequired("file")
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new migration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := migrator.Add(addConfig, addFileName); err != nil {
			return fmt.Errorf("failed to add migration: %w", err)
		}

		fmt.Printf("Migration file '%s' added successfully.\n", addFileName)
		return nil
	},
}
