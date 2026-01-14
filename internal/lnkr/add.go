package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Add adds a local file/directory to the configuration after moving it to the remote directory.
// It then creates a link from the remote location back to the local location.
func Add(path string, recursive bool, linkType string) error {
	// Normalize "symbolic" to "sym" for backward compatibility
	if linkType == "symbolic" {
		linkType = LinkTypeSymbolic
	}

	if linkType != LinkTypeHard && linkType != LinkTypeSymbolic {
		return fmt.Errorf("invalid link type: %s. Must be '%s' or '%s'", linkType, LinkTypeHard, LinkTypeSymbolic)
	}

	// Check if path is absolute
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is not allowed: %s. Please use relative path", path)
	}

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if config.Local == "" {
		return fmt.Errorf("local directory not configured. Run 'lnkr init --local <path>' first")
	}
	if config.Remote == "" {
		return fmt.Errorf("remote directory not configured. Run 'lnkr init --remote <path>' first")
	}

	// Expand paths with environment variables
	localDir, err := config.GetLocalExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand local path: %w", err)
	}
	remoteDir, err := config.GetRemoteExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand remote path: %w", err)
	}

	// Build absolute path for the local file
	localAbs := filepath.Join(localDir, path)
	fi, err := os.Stat(localAbs)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", localAbs)
	}
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if recursive && linkType == LinkTypeSymbolic {
		return fmt.Errorf("recursive option cannot be used with symbolic links")
	}

	// Check existing links to avoid duplicates
	existing := make(map[string]struct{})
	for _, link := range config.Links {
		existing[link.Path] = struct{}{}
	}

	var targets []string

	// Add paths based on type and recursive flag
	if fi.IsDir() {
		if linkType == LinkTypeHard && !recursive {
			return fmt.Errorf("recursive option must be set when adding a directory with hard links")
		}

		if linkType == LinkTypeHard {
			// Walk directory and add all files for hard links
			err := filepath.Walk(localAbs, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return addPathToTargets(p, localDir, existing, &targets)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}
		} else {
			// Add directory itself for symbolic links
			if err := addPathToTargets(localAbs, localDir, existing, &targets); err != nil {
				return err
			}
		}
	} else {
		// Add single file
		if err := addPathToTargets(localAbs, localDir, existing, &targets); err != nil {
			return err
		}
	}

	if len(targets) == 0 {
		fmt.Println("No new paths to add.")
		return nil
	}

	// Move files from local to remote and create links
	for _, t := range targets {
		localPath := filepath.Join(localDir, t)
		remotePath := filepath.Join(remoteDir, t)

		// Create parent directory in remote if needed
		remoteParentDir := filepath.Dir(remotePath)
		if err := os.MkdirAll(remoteParentDir, 0755); err != nil {
			return fmt.Errorf("failed to create remote directory %s: %w", remoteParentDir, err)
		}

		// Move the file/directory from local to remote
		if err := os.Rename(localPath, remotePath); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", localPath, remotePath, err)
		}
		fmt.Printf("Moved: %s -> %s\n", localPath, remotePath)

		// Create link from remote to local
		if err := createLink(remotePath, localPath, linkType); err != nil {
			// Try to restore by moving back
			if restoreErr := os.Rename(remotePath, localPath); restoreErr != nil {
				fmt.Printf("Warning: failed to restore %s: %v\n", localPath, restoreErr)
			}
			return fmt.Errorf("failed to create link for %s: %w", t, err)
		}

		// Add to config
		config.Links = append(config.Links, Link{Path: t, Type: linkType})
		fmt.Printf("Added link: %s (type: %s)\n", t, linkType)
	}

	sort.Slice(config.Links, func(i, j int) bool {
		return config.Links[i].Path < config.Links[j].Path
	})

	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Apply all configured link paths to GitExclude
	if err := applyAllLinksToGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to apply link paths to GitExclude: %v\n", err)
	}

	return nil
}

// createLink creates a link from source to target
func createLink(source, target, linkType string) error {
	switch linkType {
	case LinkTypeHard:
		if err := os.Link(source, target); err != nil {
			return fmt.Errorf("failed to create hard link: %w", err)
		}
		fmt.Printf("Created hard link: %s -> %s\n", target, source)
	case LinkTypeSymbolic:
		if err := os.Symlink(source, target); err != nil {
			return fmt.Errorf("failed to create symbolic link: %w", err)
		}
		fmt.Printf("Created symbolic link: %s -> %s\n", target, source)
	default:
		return fmt.Errorf("unknown link type: %s", linkType)
	}
	return nil
}

// addPathToTargets adds a single path to the targets slice if it doesn't already exist
func addPathToTargets(absPath, baseDir string, existing map[string]struct{}, targets *[]string) error {
	relPath, err := filepath.Rel(baseDir, absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}
	if _, ok := existing[relPath]; !ok {
		*targets = append(*targets, relPath)
	}
	return nil
}
