package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Create links based on .lnkr.toml configuration",
	Long: `Create hard links or symbolic links based on the .lnkr.toml configuration file.

Links are created from the remote directory (source) to the local directory.
Already-linked entries are skipped, so the command can be re-run safely.
A local file that exists but is not a link to remote is reported as an error.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return lnkr.CreateLinks()
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
