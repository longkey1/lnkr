package cmd

import (
	"fmt"

	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Move a file/directory to remote and link it back to local",
	Long: `Move a local file/directory to the remote directory and replace it with a
link pointing back to the moved file. The entry is recorded in .lnkr.toml.

The path may be absolute or relative to the current directory, as long as it
is inside the local directory.

This command will:
- Move the specified local file/directory to the remote directory
- Create a link from remote to local
- Add the entry to .lnkr.toml configuration
- If recursive flag is set with hard links, it will also add all files in the directory`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recursive, _ := cmd.Flags().GetBool("recursive")
		linkTypeFlag, _ := cmd.Flags().GetString("type")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		path := args[0]

		// Load config to get default link type
		config, err := lnkr.LoadConfigForCLI()
		if err != nil {
			return err
		}

		// Determine link type: use flag if explicitly set, otherwise use config default
		linkType := config.GetLinkType()
		if linkTypeFlag != "" {
			if linkTypeFlag != lnkr.LinkTypeSymbolic && linkTypeFlag != lnkr.LinkTypeHard && linkTypeFlag != "symbolic" {
				return fmt.Errorf("invalid link type %q. Must be 'sym' or 'hard'", linkTypeFlag)
			}
			// Normalize "symbolic" to "sym"
			if linkTypeFlag == "symbolic" {
				linkType = lnkr.LinkTypeSymbolic
			} else {
				linkType = linkTypeFlag
			}
		}

		return lnkr.Add(path, recursive, linkType, dryRun)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Add flags
	addCmd.Flags().BoolP("recursive", "r", false, "Add recursively (include all files in directory, for hard links)")
	addCmd.Flags().StringP("type", "t", "", "Link type: 'sym' or 'hard' (default: config setting or sym)")
	addCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
}
