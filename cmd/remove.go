package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Remove a link from the project",
	Long:  `Remove a link (and its subdirectories) from the .lnkr.toml configuration by path.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return lnkr.Remove(args[0])
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
