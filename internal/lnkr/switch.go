package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Switch changes the link type of an existing entry.
// If newType is empty, it toggles between sym and hard.
func Switch(path string, newType string) error {
	// Normalize "symbolic" to "sym" for backward compatibility
	if newType == "symbolic" {
		newType = LinkTypeSymbolic
	}

	// Validate newType if provided
	if newType != "" && newType != LinkTypeSymbolic && newType != LinkTypeHard {
		return fmt.Errorf("invalid link type: %s. Must be '%s' or '%s'", newType, LinkTypeSymbolic, LinkTypeHard)
	}

	// Check if path is absolute
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is not allowed: %s. Please use relative path", path)
	}

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if config.Local == "" || config.Remote == "" {
		return fmt.Errorf("local or remote directory not configured. Run 'lnkr init' first")
	}

	// Find the entry (exact match or prefix match for hard-linked directories)
	var targetIndex int = -1
	var isHardLinkedDir bool
	for i, link := range config.Links {
		if link.Path == path {
			targetIndex = i
			break
		}
	}

	// If not found, check if this is a hard-linked directory (files under path exist)
	if targetIndex == -1 {
		pathPrefix := path + string(os.PathSeparator)
		for _, link := range config.Links {
			if strings.HasPrefix(link.Path, pathPrefix) {
				isHardLinkedDir = true
				break
			}
		}
		if !isHardLinkedDir {
			return fmt.Errorf("path not found in configuration: %s", path)
		}
	}

	var currentType string
	if isHardLinkedDir {
		currentType = LinkTypeHard
	} else {
		currentType = config.Links[targetIndex].Type
		if currentType == "" {
			currentType = LinkTypeSymbolic
		}
	}

	// Determine new type
	targetType := newType
	if targetType == "" {
		// Toggle
		if currentType == LinkTypeSymbolic {
			targetType = LinkTypeHard
		} else {
			targetType = LinkTypeSymbolic
		}
	}

	// Check if change is needed
	if currentType == targetType {
		fmt.Printf("Link type is already %s: %s\n", targetType, path)
		return nil
	}

	localPath := filepath.Join(config.Local, path)
	remotePath := filepath.Join(config.Remote, path)

	// Check if this is a directory
	fi, err := os.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote path: %w", err)
	}

	if fi.IsDir() || isHardLinkedDir {
		// Handle directory conversion
		return switchDirectory(config, targetIndex, path, currentType, targetType, isHardLinkedDir)
	}

	// Handle file conversion
	return switchFile(config, targetIndex, path, localPath, remotePath, currentType, targetType)
}

// switchFile switches a single file's link type
func switchFile(config *Config, targetIndex int, path, localPath, remotePath, currentType, targetType string) error {
	// Remove existing link
	if err := os.Remove(localPath); err != nil {
		return fmt.Errorf("failed to remove existing link: %w", err)
	}

	// Create new link
	if err := createLink(remotePath, localPath, targetType); err != nil {
		// Try to restore old link
		if restoreErr := createLink(remotePath, localPath, currentType); restoreErr != nil {
			fmt.Printf("Warning: failed to restore original link: %v\n", restoreErr)
		}
		return fmt.Errorf("failed to create new link: %w", err)
	}

	// Update config
	config.Links[targetIndex].Type = targetType
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Switched link type: %s -> %s for %s\n", currentType, targetType, path)
	return nil
}

// switchDirectory handles directory link type conversion
func switchDirectory(config *Config, targetIndex int, path, currentType, targetType string, isHardLinkedDir bool) error {
	localPath := filepath.Join(config.Local, path)
	remotePath := filepath.Join(config.Remote, path)

	if targetType == LinkTypeHard {
		// sym -> hard: Remove symlink dir, create hard links for each file
		if err := os.Remove(localPath); err != nil {
			return fmt.Errorf("failed to remove symlink directory: %w", err)
		}

		// Walk remote directory and create hard links for each file
		var newLinks []Link
		err := filepath.Walk(remotePath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				relPath, _ := filepath.Rel(remotePath, p)
				localDir := filepath.Join(localPath, relPath)
				if relPath == "." {
					localDir = localPath
				}
				return os.MkdirAll(localDir, 0755)
			}

			// Create hard link for file
			relPath, _ := filepath.Rel(config.Remote, p)
			localFile := filepath.Join(config.Local, relPath)
			if err := createLink(p, localFile, LinkTypeHard); err != nil {
				return err
			}
			newLinks = append(newLinks, Link{Path: relPath, Type: LinkTypeHard})
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to convert directory to hard links: %w", err)
		}

		// Remove original directory entry and add new file entries
		config.Links = append(config.Links[:targetIndex], config.Links[targetIndex+1:]...)
		config.Links = append(config.Links, newLinks...)

	} else {
		// hard -> sym: Simply remove hard links and create symlink dir
		pathPrefix := path + string(os.PathSeparator)
		var remainingLinks []Link
		for _, link := range config.Links {
			if link.Path == path || strings.HasPrefix(link.Path, pathPrefix) {
				os.Remove(filepath.Join(config.Local, link.Path))
			} else {
				remainingLinks = append(remainingLinks, link)
			}
		}

		// Remove local directory tree and create symbolic link
		os.RemoveAll(localPath)
		if err := createLink(remotePath, localPath, LinkTypeSymbolic); err != nil {
			return fmt.Errorf("failed to create symbolic link: %w", err)
		}

		remainingLinks = append(remainingLinks, Link{Path: path, Type: LinkTypeSymbolic})
		config.Links = remainingLinks
	}

	sort.Slice(config.Links, func(i, j int) bool {
		return config.Links[i].Path < config.Links[j].Path
	})

	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Switched link type: %s -> %s for %s (recursive)\n", currentType, targetType, path)
	return nil
}
