package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of links in .lnkr.toml configuration",
	Long:  `Show the status of all links defined in the .lnkr.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return lnkr.Status()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
