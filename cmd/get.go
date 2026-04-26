package cmd

import (
	"fmt"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a secret's value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		store := storage.NewStore("secrets.json", cfg.MasterPassword)
		if err := store.Load(); err != nil {
			return err
		}

		sec, found := store.Secrets[key]
		if !found {
			return fmt.Errorf("secret for '%s' not found", key)
		}

		fmt.Printf("Secret for '%s': %s\n", key, sec.Value)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
