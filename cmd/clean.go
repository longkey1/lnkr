package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove .lnkr.toml and its git exclude entries",
	Long: `Clean up files and changes made by the init command.

This command will:
- Remove the .lnkr.toml configuration file if it exists
- Remove the LNKR section from the git exclude file

It does not remove the links themselves; run 'lnkr unlink' first if links
are still in place.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")
		return lnkr.Clean(dryRun, yes)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	cleanCmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")
}
