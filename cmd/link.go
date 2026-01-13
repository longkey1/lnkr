package cmd

import (
	"fmt"
	"os"

	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Create links based on .lnkr.toml configuration",
	Long:  `Create hard links or symbolic links based on the .lnkr.toml configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := lnkr.CreateLinks(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
