package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Create links based on .lnkr.toml configuration",
	Long:  `Create hard links or symbolic links based on the .lnkr.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return lnkr.CreateLinks()
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
