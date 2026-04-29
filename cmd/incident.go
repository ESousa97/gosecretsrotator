package cmd

import (
	"fmt"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/rotation"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var incidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Handle security incidents and rollbacks",
}

var rollbackSecretName string

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback a secret to its previous successful version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if rollbackSecretName == "" {
			return fmt.Errorf("secret name is required via --secret-name")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		hdb, err := storage.NewHistoryDB("history.db")
		if err != nil {
			return err
		}
		defer func() {
			if err := hdb.Close(); err != nil {
				fmt.Printf("failed to close history db: %v\n", err)
			}
		}()

		store := storage.NewStore("secrets.json", cfg.MasterPassword)
		if err := store.Load(); err != nil {
			return err
		}

		if err := rotation.RollbackSecret(store, hdb, rollbackSecretName); err != nil {
			return err
		}

		fmt.Printf("Successfully rolled back secret '%s' to previous version\n", rollbackSecretName)
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVar(&rollbackSecretName, "secret-name", "", "Name of the secret to rollback")
	incidentCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(incidentCmd)
}
