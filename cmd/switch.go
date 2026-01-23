package cmd

import (
	"fmt"

	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <path> [sym|hard]",
	Short: "Switch link type for an entry",
	Long: `Switch the link type of an existing entry between sym and hard.

If no type is specified, it toggles between sym and hard.
Note: Directories cannot be converted to hard links.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		var linkType string
		if len(args) > 1 {
			linkType = args[1]
			// Normalize "symbolic" to "sym" for backward compatibility
			if linkType == "symbolic" {
				linkType = lnkr.LinkTypeSymbolic
			}
			if linkType != lnkr.LinkTypeSymbolic && linkType != lnkr.LinkTypeHard {
				return fmt.Errorf("invalid link type %q. Must be 'sym' or 'hard'", linkType)
			}
		}

		return lnkr.Switch(path, linkType)
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
