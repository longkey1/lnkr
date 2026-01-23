package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up files created by init command",
	Long: `Clean up files and changes made by the init command.

This command will:
- Remove .lnkr.toml configuration file if it exists
- Remove .lnkr.toml entry from .git/info/exclude`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return lnkr.Clean()
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
