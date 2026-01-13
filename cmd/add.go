package cmd

import (
	"fmt"
	"os"

	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Add a link to the project",
	Long: `Add a link to the project configuration.

This command will:
- Add the specified path as a link in the .lnkr.toml configuration
- If recursive flag is set, it will also add all subdirectories and files
- Update the configuration file with the new link entries`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		recursive, _ := cmd.Flags().GetBool("recursive")
		symbolic, _ := cmd.Flags().GetBool("symbolic")
		symbolicChanged := cmd.Flags().Changed("symbolic")
		fromRemote, _ := cmd.Flags().GetBool("from-remote")
		path := args[0]

		// Load config to get default link type
		config, err := lnkr.LoadConfigForCLI()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Determine link type: use flag if explicitly set, otherwise use config default
		linkType := config.GetLinkType()
		if symbolicChanged {
			if symbolic {
				linkType = lnkr.LinkTypeSymbolic
			} else {
				linkType = lnkr.LinkTypeHard
			}
		}

		if err := lnkr.Add(path, recursive, linkType, fromRemote); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Add flags
	addCmd.Flags().BoolP("recursive", "r", false, "Add recursively (include subdirectories and files)")
	addCmd.Flags().BoolP("symbolic", "s", false, "Create symbolic link (overrides link_type in config)")
	addCmd.Flags().Bool("from-remote", false, "Use remote directory as base for relative paths")
}
