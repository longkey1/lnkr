package cmd

import (
	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Remove links based on .lnkr.toml configuration",
	Long:  `Remove hard links, symbolic links, or directories based on the .lnkr.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return lnkr.Unlink()
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}
