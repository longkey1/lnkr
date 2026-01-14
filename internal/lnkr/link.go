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

	for _, link := range config.Links {
		if err := createLinkEntry(link, config); err != nil {
			fmt.Printf("Error creating link for %s: %v\n", link.Path, err)
			continue
		}
	}

	// Apply all link paths to GitExclude
	if err := applyAllLinksToGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to apply link paths to GitExclude: %v\n", err)
	}

	fmt.Println("Link creation completed.")
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
			return fmt.Errorf("hard links cannot be created for directories: %s", sourceAbs)
		}
		if err := os.Link(sourceAbs, targetAbs); err != nil {
			return fmt.Errorf("failed to create hard link: %w", err)
		}
		fmt.Printf("Created hard link: %s -> %s\n", targetAbs, sourceAbs)
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
