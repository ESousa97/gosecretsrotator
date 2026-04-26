package cmd

import (
	"github.com/spf13/cobra"
)

var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject secrets into different providers (docker, file, etc.)",
}

func init() {
	rootCmd.AddCommand(injectCmd)
}
