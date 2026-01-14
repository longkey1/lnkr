package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Remove removes a link from the configuration and restores the file from remote to local.
// This is the reverse operation of Add.
func Remove(path string) error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
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

	// Find matching links
	var linksToRemove []Link
	var newLinks []Link
	for _, link := range config.Links {
		if link.Path == path || strings.HasPrefix(link.Path, path+string(os.PathSeparator)) {
			linksToRemove = append(linksToRemove, link)
		} else {
			newLinks = append(newLinks, link)
		}
	}

	if len(linksToRemove) == 0 {
		fmt.Println("No matching links found to remove.")
		return nil
	}

	// Sort links to remove in reverse order (deepest paths first)
	// This ensures child files are processed before parent directories
	sort.Slice(linksToRemove, func(i, j int) bool {
		return linksToRemove[i].Path > linksToRemove[j].Path
	})

	// Process each link: remove link, move file from remote to local
	for _, link := range linksToRemove {
		if err := restoreFromRemote(link, localDir, remoteDir); err != nil {
			return fmt.Errorf("failed to restore %s: %w", link.Path, err)
		}
		fmt.Printf("Removed link: %s\n", link.Path)
	}

	// Sort remaining links
	sort.Slice(newLinks, func(i, j int) bool {
		return newLinks[i].Path < newLinks[j].Path
	})

	config.Links = newLinks
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Apply all remaining link paths to GitExclude
	if err := applyAllLinksToGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to apply link paths to GitExclude: %v\n", err)
	}

	return nil
}

// restoreFromRemote removes the link at local and moves the file from remote back to local
func restoreFromRemote(link Link, localDir, remoteDir string) error {
	localPath := filepath.Join(localDir, link.Path)
	remotePath := filepath.Join(remoteDir, link.Path)

	// Check if remote file exists
	if _, err := os.Stat(remotePath); os.IsNotExist(err) {
		return fmt.Errorf("remote file does not exist: %s", remotePath)
	}

	// Remove the link at local path
	switch link.Type {
	case LinkTypeHard:
		// Hard link: just remove the local file (it's a hard link to remote)
		if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove hard link at %s: %w", localPath, err)
		}
	case LinkTypeSymbolic:
		// Symbolic link: remove the symlink
		fi, err := os.Lstat(localPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat %s: %w", localPath, err)
		}
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(localPath); err != nil {
					return fmt.Errorf("failed to remove symbolic link at %s: %w", localPath, err)
				}
			} else {
				return fmt.Errorf("expected symbolic link at %s but found regular file", localPath)
			}
		}
	default:
		return fmt.Errorf("unknown link type: %s", link.Type)
	}

	// Create parent directory in local if needed
	localParentDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localParentDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory %s: %w", localParentDir, err)
	}

	// Move the file from remote to local
	if err := os.Rename(remotePath, localPath); err != nil {
		return fmt.Errorf("failed to move %s to %s: %w", remotePath, localPath, err)
	}
	fmt.Printf("Restored: %s -> %s\n", remotePath, localPath)

	// Clean up empty parent directories in remote
	cleanEmptyDirs(filepath.Dir(remotePath), remoteDir)

	return nil
}

// cleanEmptyDirs removes empty directories from path up to (but not including) stopAt
func cleanEmptyDirs(path, stopAt string) {
	for path != stopAt && path != "." && path != "/" {
		entries, err := os.ReadDir(path)
		if err != nil || len(entries) > 0 {
			break
		}
		if err := os.Remove(path); err != nil {
			break
		}
		path = filepath.Dir(path)
	}
}
