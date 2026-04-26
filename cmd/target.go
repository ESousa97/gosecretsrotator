package cmd

import (
	"fmt"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "Manage rotation targets attached to a secret",
}

var targetAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Attach a target (docker|file) to a secret",
}

var (
	tDockerSecret    string
	tDockerContainer string
	tDockerEnvKey    string
)

var targetAddDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Attach a Docker container env var as a rotation target",
	RunE: func(cmd *cobra.Command, args []string) error {
		return appendTarget(tDockerSecret, storage.Target{
			Type:      "docker",
			Container: tDockerContainer,
			EnvKey:    tDockerEnvKey,
		})
	},
}

var (
	tFileSecret string
	tFilePath   string
	tFileKey    string
)

var targetAddFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Attach a .env or .yaml field as a rotation target",
	RunE: func(cmd *cobra.Command, args []string) error {
		return appendTarget(tFileSecret, storage.Target{
			Type:    "file",
			Path:    tFilePath,
			FileKey: tFileKey,
		})
	},
}

func appendTarget(secretKey string, t storage.Target) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	store := storage.NewStore("secrets.json", cfg.MasterPassword)
	if err := store.Load(); err != nil {
		return err
	}
	sec, ok := store.Secrets[secretKey]
	if !ok {
		return fmt.Errorf("secret '%s' not found", secretKey)
	}
	sec.Targets = append(sec.Targets, t)
	if err := store.Save(); err != nil {
		return err
	}
	fmt.Printf("Target %+v attached to '%s'\n", t, secretKey)
	return nil
}

func init() {
	targetAddDockerCmd.Flags().StringVarP(&tDockerSecret, "secret", "s", "", "Secret key in the vault")
	targetAddDockerCmd.Flags().StringVarP(&tDockerContainer, "container", "c", "", "Target Docker container name")
	targetAddDockerCmd.Flags().StringVarP(&tDockerEnvKey, "env-key", "k", "", "Env var name inside the container")
	_ = targetAddDockerCmd.MarkFlagRequired("secret")
	_ = targetAddDockerCmd.MarkFlagRequired("container")
	_ = targetAddDockerCmd.MarkFlagRequired("env-key")

	targetAddFileCmd.Flags().StringVarP(&tFileSecret, "secret", "s", "", "Secret key in the vault")
	targetAddFileCmd.Flags().StringVarP(&tFilePath, "path", "p", "", "Path to .env or .yaml file")
	targetAddFileCmd.Flags().StringVarP(&tFileKey, "field", "f", "", "Field name to update in the file")
	_ = targetAddFileCmd.MarkFlagRequired("secret")
	_ = targetAddFileCmd.MarkFlagRequired("path")
	_ = targetAddFileCmd.MarkFlagRequired("field")

	targetAddCmd.AddCommand(targetAddDockerCmd, targetAddFileCmd)
	targetCmd.AddCommand(targetAddCmd)
	rootCmd.AddCommand(targetCmd)
}
