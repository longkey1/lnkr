package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Unlink() error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(config.Links) == 0 {
		fmt.Printf("No links found in %s\n", ConfigFileName)
		return nil
	}

	// Use local directory as base for resolving link paths
	baseDir := config.Local

	for _, link := range config.Links {
		if err := removeLinkWithBase(link, baseDir); err != nil {
			fmt.Printf("Error removing link for %s: %v\n", link.Path, err)
			continue
		}
	}

	// Remove all link paths from GitExclude
	if err := removeAllLinksFromGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to remove link paths from GitExclude: %v\n", err)
	}

	fmt.Println("Link removal completed.")
	return nil
}

func removeLinkWithBase(link Link, baseDir string) error {
	// Resolve absolute path for link
	linkAbs := filepath.Join(baseDir, link.Path)

	if _, err := os.Stat(linkAbs); os.IsNotExist(err) {
		fmt.Printf("Path does not exist, skipping: %s\n", linkAbs)
		return nil
	}

	switch link.Type {
	case LinkTypeHard:
		info, err := os.Stat(linkAbs)
		if err != nil {
			return fmt.Errorf("failed to stat path: %w", err)
		}

		if info.IsDir() {
			if err := os.RemoveAll(linkAbs); err != nil {
				return fmt.Errorf("failed to remove directory: %w", err)
			}
			fmt.Printf("Removed directory: %s\n", linkAbs)
		} else {
			if err := os.Remove(linkAbs); err != nil {
				return fmt.Errorf("failed to remove hard link: %w", err)
			}
			fmt.Printf("Removed hard link: %s\n", linkAbs)
		}
	case LinkTypeSymbolic:
		if err := os.Remove(linkAbs); err != nil {
			return fmt.Errorf("failed to remove symbolic link: %w", err)
		}
		fmt.Printf("Removed symbolic link: %s\n", linkAbs)
	default:
		return fmt.Errorf("unknown link type: %s", link.Type)
	}

	return nil
}

// removeAllLinksFromGitExclude removes all configured link paths from GitExclude
func removeAllLinksFromGitExclude(config *Config) error {
	if len(config.Links) == 0 {
		return nil
	}

	excludePath := config.GetGitExcludePath()

	// Check if exclude file exists
	if _, err := os.Stat(excludePath); os.IsNotExist(err) {
		return nil
	}

	// Read existing content
	content, err := os.ReadFile(excludePath)
	if err != nil {
		return err
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")

	// Find section boundaries
	sectionStart := -1
	sectionEnd := -1
	sectionMarker := GitExcludeSectionStart
	endMarker := GitExcludeSectionEnd

	for i, line := range lines {
		if strings.TrimSpace(line) == sectionMarker {
			sectionStart = i
		}
		if sectionStart != -1 && strings.TrimSpace(line) == endMarker {
			sectionEnd = i
			break
		}
	}

	// If section doesn't exist, nothing to remove
	if sectionStart == -1 || sectionEnd == -1 {
		return nil
	}

	// Remove the entire section
	newLines := append(lines[:sectionStart], lines[sectionEnd+1:]...)

	// Write back the content
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(excludePath, []byte(newContent), 0644); err != nil {
		return err
	}

	fmt.Printf("Removed all link paths from %s\n", excludePath)
	return nil
}
