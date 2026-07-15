package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Remove the links at local (entries and remote files are kept)",
	Long: `Remove the links defined in .lnkr.toml from the local directory.

The entries in .lnkr.toml and the files in the remote directory are kept, so
'lnkr link' can re-create the links later. For hard-linked directories, files
that are not linked to remote (e.g. added after linking) are kept.

To restore files back to local instead, use 'lnkr remove'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")
		return lnkr.Unlink(dryRun, yes)
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
	unlinkCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	unlinkCmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")
}
