package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the project",
	Long: `Initialize the project by creating necessary configuration files and setting up git exclusions.

This command will:
- Create .lnkr.toml configuration file if it doesn't exist
- Add .lnkr.toml to .git/info/exclude to prevent it from being tracked`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Get local root and remote root from global config (env var > config file > default)
		// Expand environment variables in the paths
		localRoot, err := lnkr.ExpandPath(lnkr.GetLocalRoot())
		if err != nil {
			return fmt.Errorf("failed to expand local root: %w", err)
		}
		remoteRoot, err := lnkr.ExpandPath(lnkr.GetRemoteRoot())
		if err != nil {
			return fmt.Errorf("failed to expand remote root: %w", err)
		}

		// Check if remote root directory exists
		if info, err := os.Stat(remoteRoot); os.IsNotExist(err) {
			return fmt.Errorf("remote root directory does not exist: %s", remoteRoot)
		} else if err != nil {
			return fmt.Errorf("failed to stat remote root directory: %w", err)
		} else if !info.IsDir() {
			return fmt.Errorf("remote root path is not a directory: %s", remoteRoot)
		}

		// Get remote directory from flag or default
		remoteDir, _ := cmd.Flags().GetString("remote")
		if remoteDir == "" {
			// Use lnkr package function to get default remote path
			remoteDir = lnkr.GetDefaultRemotePath(currentDir, localRoot, remoteRoot)
		} else {
			// If remoteDir is specified, make it absolute path based on remoteRoot
			if !filepath.IsAbs(remoteDir) {
				remoteDir = filepath.Join(remoteRoot, remoteDir)
			}
		}

		// Set default git exclude path if not specified (env var > config file > default)
		gitExcludePath, _ := cmd.Flags().GetString("git-exclude-path")
		if gitExcludePath == "" {
			gitExcludePath = lnkr.GetGlobalGitExcludePath()
		}

		return lnkr.Init(remoteDir, gitExcludePath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("remote", "r", "", "Remote directory to save in .lnkr.toml (if not specified, uses remote_root + relative path from local_root)")
	initCmd.Flags().String("git-exclude-path", "", "Custom path for git exclude file (default: .git/info/exclude)")
}
