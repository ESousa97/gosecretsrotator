package cmd

import (
	"fmt"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [key] [value]",
	Short: "Add or update a secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		store := storage.NewStore("secrets.json", cfg.MasterPassword)
		if err := store.Load(); err != nil {
			return err
		}

		if store.Secrets == nil {
			store.Secrets = make(map[string]string)
		}
		store.Secrets[key] = value

		if err := store.Save(); err != nil {
			return err
		}

		fmt.Printf("Secret for '%s' saved successfully.\n", key)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
