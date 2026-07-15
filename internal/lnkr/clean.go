package lnkr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Clean removes the configuration file and its git exclude entries.
// It does not touch the links themselves; run 'lnkr unlink' first.
func Clean(dryRun, assumeYes bool) error {
	config, err := loadConfig()
	configExists := true
	if err != nil {
		if !errors.Is(err, ErrConfigNotFound) {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		configExists = false
		config = &Config{}
	}
	excludePath := config.GetGitExcludePath()
	configPath := ConfigFileName
	if config.dir != "" {
		configPath = filepath.Join(config.dir, ConfigFileName)
	}

	if dryRun {
		if configExists {
			if len(config.Links) > 0 {
				fmt.Printf("Warning: %d link(s) are still registered in %s\n", len(config.Links), configPath)
			}
			fmt.Printf("Would remove %s\n", configPath)
		}
		fmt.Printf("Would remove LNKR entries from %s\n", excludePath)
		return nil
	}

	if configExists {
		if len(config.Links) > 0 {
			fmt.Printf("Warning: %d link(s) are still registered in %s; run 'lnkr unlink' first to remove the links themselves\n", len(config.Links), configPath)
		}
		if !assumeYes && !confirm(fmt.Sprintf("Remove %s and its entries in %s?", configPath, excludePath)) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Remove .lnkr.toml file if it exists
	if err := removeLnkToml(configPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", configPath, err)
	}

	// Remove the LNKR section, and any plain entry left by old versions
	if removed, err := removeGitExcludeSection(excludePath); err != nil {
		return fmt.Errorf("failed to remove LNKR section from %s: %w", excludePath, err)
	} else if removed {
		fmt.Printf("Removed LNKR section from %s\n", excludePath)
	}
	if err := removeFromGitExcludeWithPath(excludePath, ConfigFileName); err != nil {
		return fmt.Errorf("failed to remove from %s: %w", excludePath, err)
	}

	fmt.Println("Cleanup completed successfully!")
	return nil
}

// removeLnkToml removes the configuration file if it exists
func removeLnkToml(filename string) error {
	// Check if file exists (Lstat so a broken symlink is still removed)
	if _, err := os.Lstat(filename); os.IsNotExist(err) {
		fmt.Printf("%s does not exist\n", filename)
		return nil
	}

	// Remove file
	if err := os.Remove(filename); err != nil {
		return err
	}

	fmt.Printf("Removed %s\n", filename)
	return nil
}

// removeFromGitExcludeWithPath removes plain (non-section) entries from a git exclude file
func removeFromGitExcludeWithPath(excludePath, entry string) error {
	// Check if exclude file exists
	if _, err := os.Stat(excludePath); os.IsNotExist(err) {
		fmt.Printf("%s does not exist\n", excludePath)
		return nil
	}

	// Read existing content
	content, err := os.ReadFile(excludePath)
	if err != nil {
		return err
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")

	// Check if entry exists
	entryExists := false
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			entryExists = true
			break
		}
	}

	if !entryExists {
		fmt.Printf("%s does not exist in %s\n", entry, excludePath)
		return nil
	}

	// Filter out the entry
	var newLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != entry {
			newLines = append(newLines, line)
		}
	}

	// Write back the filtered content
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(excludePath, []byte(newContent), 0644); err != nil {
		return err
	}

	fmt.Printf("Removed %s from %s\n", entry, excludePath)
	return nil
}
