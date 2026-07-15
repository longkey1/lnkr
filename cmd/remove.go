package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Remove a link entry and restore the file from remote to local",
	Long: `Remove a link (and its subdirectories) from the .lnkr.toml configuration.
The link at local is removed and the file is moved back from remote to local.
This is the reverse operation of 'add'.

To remove only the links while keeping the entries and remote files, use
'lnkr unlink' instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return lnkr.Remove(args[0], dryRun)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
}
