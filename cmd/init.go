package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/longkey1/lnkr/internal/lnkr"
	"github.com/spf13/cobra"
)

var (
	remoteDir      string
	gitExcludePath string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the project",
	Long: `Initialize the project by creating necessary configuration files and setting up git exclusions.

This command will:
- Create .lnkr.toml configuration file if it doesn't exist
- Add .lnkr.toml to .git/info/exclude to prevent it from being tracked`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
			os.Exit(1)
		}

		// Get local root and remote root from global config (env var > config file > default)
		// Expand environment variables in the paths
		localRoot, err := lnkr.ExpandPath(lnkr.GetLocalRoot())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to expand local root: %v\n", err)
			os.Exit(1)
		}
		remoteRoot, err := lnkr.ExpandPath(lnkr.GetRemoteRoot())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to expand remote root: %v\n", err)
			os.Exit(1)
		}

		// Check if remote root directory exists
		if info, err := os.Stat(remoteRoot); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: remote root directory does not exist: %s\n", remoteRoot)
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to stat remote root directory: %v\n", err)
			os.Exit(1)
		} else if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: remote root path is not a directory: %s\n", remoteRoot)
			os.Exit(1)
		}

		// Get remote directory from flag or default
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
		if gitExcludePath == "" {
			gitExcludePath = lnkr.GetGlobalGitExcludePath()
		}

		if err := lnkr.Init(remoteDir, gitExcludePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&remoteDir, "remote", "r", "", "Remote directory to save in .lnkr.toml (if not specified, uses remote_root + relative path from local_root)")
	initCmd.Flags().StringVar(&gitExcludePath, "git-exclude-path", "", "Custom path for git exclude file (default: .git/info/exclude)")
}
