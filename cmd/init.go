package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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

		// Get the number of depth to go up from environment variable, default to DefaultRemoteDepth
		depthStr := os.Getenv("LNKR_REMOTE_DEPTH")
		depth := lnkr.DefaultRemoteDepth // default value
		if depthStr != "" {
			if parsedDepth, err := strconv.Atoi(depthStr); err == nil && parsedDepth > 0 {
				depth = parsedDepth
			}
		}

		// Get base directory for remote
		baseDir := os.Getenv("LNKR_REMOTE_ROOT")
		if baseDir == "" {
			// Default to $HOME/.config/lnkr
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to get home directory: %v\n", err)
				os.Exit(1)
			}
			baseDir = filepath.Join(homeDir, ".config", "lnkr")
		}

		// Check if base directory exists
		if info, err := os.Stat(baseDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: base directory does not exist: %s\n", baseDir)
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to stat base directory: %v\n", err)
			os.Exit(1)
		} else if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: base path is not a directory: %s\n", baseDir)
			os.Exit(1)
		}

		// Get remote directory from flag or default
		if remoteDir == "" {
			// Use lnkr package function to get default remote path
			remoteDir = lnkr.GetDefaultRemotePath(currentDir, baseDir, depth)
		} else {
			// If remoteDir is specified, make it absolute path based on baseDir
			if !filepath.IsAbs(remoteDir) {
				remoteDir = filepath.Join(baseDir, remoteDir)
			}
		}

		// Set default git exclude path if not specified
		if gitExcludePath == "" {
			gitExcludePath = lnkr.GitExcludePath
		}

		if err := lnkr.Init(remoteDir, gitExcludePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&remoteDir, "remote", "r", "", "Remote directory to save in .lnkr.toml (if not specified, uses LNKR_REMOTE_ROOT/project-name or parent-dir/current-dir based on LNKR_REMOTE_DEPTH)")
	initCmd.Flags().StringVar(&gitExcludePath, "git-exclude-path", "", "Custom path for git exclude file (default: .git/info/exclude)")
}
