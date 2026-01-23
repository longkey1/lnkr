package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/longkey1/lnkr/internal/version"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lnkr",
	Short: "A link helper CLI tool",
	Long: `lnkr is a command line tool for managing and working with links.
It provides various utilities for link manipulation, validation, and management.`,
	Version:       version.GetVersion(),
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Initialize global configuration (viper)
	lnkr.InitGlobalConfig()
}
