package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateLinks creates links based on configuration.
// Links are always created from remote to local (remote is the source, local is the link).
func CreateLinks() error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(config.Links) == 0 {
		fmt.Printf("No links found in %s\n", ConfigFileName)
		return nil
	}

	var errorCount int
	for _, link := range config.Links {
		if err := createLinkEntry(link, config); err != nil {
			fmt.Printf("Error creating link for %s: %v\n", link.Path, err)
			errorCount++
			continue
		}
	}

	// Apply all link paths to GitExclude
	if err := applyAllLinksToGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to apply link paths to GitExclude: %v\n", err)
	}

	totalCount := len(config.Links)
	successCount := totalCount - errorCount
	if errorCount == 0 {
		fmt.Printf("Link creation completed. (%d/%d succeeded)\n", successCount, totalCount)
	} else if successCount == 0 {
		fmt.Printf("Link creation failed. (%d/%d failed)\n", errorCount, totalCount)
		return fmt.Errorf("all %d links failed to create", errorCount)
	} else {
		fmt.Printf("Link creation completed with errors. (%d/%d succeeded, %d failed)\n", successCount, totalCount, errorCount)
	}
	return nil
}

func createLinkEntry(link Link, config *Config) error {
	// Source is always remote, target is always local
	sourceDir, err := config.GetRemoteExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand remote path: %w", err)
	}
	targetDir, err := config.GetLocalExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand local path: %w", err)
	}

	// Resolve absolute paths for source and target
	sourceAbs := filepath.Join(sourceDir, link.Path)
	targetAbs := filepath.Join(targetDir, link.Path)

	// Check if source exists
	sourceInfo, err := os.Stat(sourceAbs)
	if os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourceAbs)
	}

	// Check if target already exists
	if _, err := os.Stat(targetAbs); err == nil {
		fmt.Printf("Warning: target already exists: %s\n", targetAbs)
		return nil // Skip this link instead of returning error
	}

	// Create parent directory if needed
	targetParentDir := filepath.Dir(targetAbs)
	if err := os.MkdirAll(targetParentDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	switch link.Type {
	case LinkTypeHard:
		if sourceInfo.IsDir() {
			// For directories, recursively create hard links for all files
			if err := createHardLinksRecursively(sourceAbs, targetAbs); err != nil {
				return fmt.Errorf("failed to create hard links for directory: %w", err)
			}
		} else {
			if err := os.Link(sourceAbs, targetAbs); err != nil {
				return fmt.Errorf("failed to create hard link: %w", err)
			}
			fmt.Printf("Created hard link: %s -> %s\n", targetAbs, sourceAbs)
		}
	case LinkTypeSymbolic:
		if err := os.Symlink(sourceAbs, targetAbs); err != nil {
			return fmt.Errorf("failed to create symbolic link: %w", err)
		}
		fmt.Printf("Created symbolic link: %s -> %s\n", targetAbs, sourceAbs)
	default:
		return fmt.Errorf("unknown link type: %s", link.Type)
	}

	return nil
}

// createHardLinksRecursively walks the source directory and creates hard links for all files
func createHardLinksRecursively(sourceDir, targetDir string) error {
	var fileCount int
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			// Create directory structure
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			return nil
		}

		// Check if target file already exists
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("Warning: target already exists: %s\n", targetPath)
			return nil
		}

		// Create hard link for file
		if err := os.Link(path, targetPath); err != nil {
			return fmt.Errorf("failed to create hard link %s -> %s: %w", targetPath, path, err)
		}
		fmt.Printf("Created hard link: %s -> %s\n", targetPath, path)
		fileCount++

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Created %d hard links for directory: %s\n", fileCount, sourceDir)
	return nil
}

// applyAllLinksToGitExclude removes existing LNKR section and applies all configured link paths to GitExclude
func applyAllLinksToGitExclude(config *Config) error {
	// First remove existing LNKR section
	if err := removeAllLinksFromGitExclude(config); err != nil {
		// Continue even if removal fails (section might not exist)
	}

	// Always include .lnkr.toml in the exclude list
	linkPaths := []string{ConfigFileName}
	for _, link := range config.Links {
		linkPaths = append(linkPaths, link.Path)
	}

	return addMultipleToGitExclude(linkPaths)
}
