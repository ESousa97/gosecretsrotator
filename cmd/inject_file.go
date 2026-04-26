package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/esousa97/gosecretsrotator/internal/config"
	"github.com/esousa97/gosecretsrotator/internal/providers/file"
	"github.com/esousa97/gosecretsrotator/internal/storage"
	"github.com/spf13/cobra"
)

var (
	filePath      string
	fileKey       string
	fileSecretKey string
)

var injectFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Inject a secret into a .env or .yaml file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		store := storage.NewStore("secrets.json", cfg.MasterPassword)
		if err := store.Load(); err != nil {
			return err
		}

		sec, found := store.Secrets[fileSecretKey]
		if !found {
			return fmt.Errorf("secret for '%s' not found", fileSecretKey)
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		var errInject error
		switch ext {
		case ".env":
			errInject = file.InjectEnv(filePath, fileKey, sec.Value)
		case ".yaml", ".yml":
			errInject = file.InjectYAML(filePath, fileKey, sec.Value)
		default:
			errInject = fmt.Errorf("unsupported file extension: %s. Use .env or .yaml", ext)
		}

		if errInject != nil {
			return errInject
		}

		fmt.Printf("Successfully injected secret '%s' into file '%s' as '%s'\n", fileSecretKey, filePath, fileKey)
		return nil
	},
}

func init() {
	injectFileCmd.Flags().StringVarP(&filePath, "path", "p", "", "Path to the .env or .yaml file")
	injectFileCmd.Flags().StringVarP(&fileKey, "key", "k", "", "Key name to update in the file")
	injectFileCmd.Flags().StringVarP(&fileSecretKey, "secret", "s", "", "Secret key from vault")
	_ = injectFileCmd.MarkFlagRequired("path")
	_ = injectFileCmd.MarkFlagRequired("key")
	_ = injectFileCmd.MarkFlagRequired("secret")

	injectCmd.AddCommand(injectFileCmd)
}
