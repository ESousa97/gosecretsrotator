package cmd

import (
	"fmt"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/rotation"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var rotationCmd = &cobra.Command{
	Use:   "rotation",
	Short: "Configure rotation policy for a secret",
}

var rotationIntervalDays int

var rotationSetCmd = &cobra.Command{
	Use:   "set [key]",
	Short: "Set rotation interval (in days) for a secret. Use 0 to disable.",
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
		sec, ok := store.Secrets[key]
		if !ok {
			return fmt.Errorf("secret '%s' not found", key)
		}
		sec.IntervalDays = rotationIntervalDays
		if err := store.Save(); err != nil {
			return err
		}
		fmt.Printf("Rotation interval for '%s' set to %d day(s)\n", key, rotationIntervalDays)
		return nil
	},
}

var rotateCmd = &cobra.Command{
	Use:   "rotate [key]",
	Short: "Manually rotate a secret now (generates new value, applies to targets)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
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
		if err := rotation.RotateSecret(store, hdb, key, cfg.WebhookURL); err != nil {
			return err
		}
		fmt.Printf("Rotated '%s' successfully\n", key)
		return nil
	},
}

func init() {
	rotationSetCmd.Flags().IntVarP(
		&rotationIntervalDays,
		"days",
		"d",
		30,
		"Rotation interval in days (0 disables auto rotation)",
	)
	rotationCmd.AddCommand(rotationSetCmd)
	rootCmd.AddCommand(rotationCmd)
	rootCmd.AddCommand(rotateCmd)
}
