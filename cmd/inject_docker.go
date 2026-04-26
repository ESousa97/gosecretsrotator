package cmd

import (
	"fmt"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/providers/docker"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var (
	dockerContainer string
	dockerEnvKey    string
	dockerSecretKey string
)

var injectDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Inject a secret into a Docker container's environment variable",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		store := storage.NewStore("secrets.json", cfg.MasterPassword)
		if err := store.Load(); err != nil {
			return err
		}

		sec, found := store.Secrets[dockerSecretKey]
		if !found {
			return fmt.Errorf("secret for '%s' not found", dockerSecretKey)
		}

		if err := docker.UpdateContainerEnv(dockerContainer, dockerEnvKey, sec.Value); err != nil {
			return err
		}

		fmt.Printf("Successfully injected secret '%s' into container '%s' as env '%s'\n", dockerSecretKey, dockerContainer, dockerEnvKey)
		return nil
	},
}

func init() {
	injectDockerCmd.Flags().StringVarP(&dockerContainer, "container", "c", "", "Target Docker container name")
	injectDockerCmd.Flags().StringVarP(&dockerEnvKey, "key", "k", "", "Environment variable name to set inside the container")
	injectDockerCmd.Flags().StringVarP(&dockerSecretKey, "secret", "s", "", "Secret key from vault")
	injectDockerCmd.MarkFlagRequired("container")
	injectDockerCmd.MarkFlagRequired("key")
	injectDockerCmd.MarkFlagRequired("secret")

	injectCmd.AddCommand(injectDockerCmd)
}
