package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Init performs the initialization tasks
func Init(remote string, gitExcludePath string) error {
	if err := createLnkTomlWithRemote(remote, gitExcludePath); err != nil {
		return fmt.Errorf("failed to create %s: %w", ConfigFileName, err)
	}

	if err := addToGitExclude(); err != nil {
		return fmt.Errorf("failed to add to %s: %w", GitExcludePath, err)
	}

	fmt.Println("Project initialized successfully!")
	return nil
}

// createLnkTomlWithRemote creates the .lnkr.toml file with remote if it doesn't exist
func createLnkTomlWithRemote(remote string, gitExcludePath string) error {
	filename := ConfigFileName

	// Get current directory as absolute path for local
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Convert remote to absolute path if provided
	if remote != "" {
		if !filepath.IsAbs(remote) {
			remote, err = filepath.Abs(remote)
			if err != nil {
				return fmt.Errorf("failed to convert remote to absolute path: %w", err)
			}
		}
		// remoteがディレクトリであることを保証
		info, err := os.Stat(remote)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(remote, 0755); err != nil {
				return fmt.Errorf("failed to create remote directory: %w", err)
			}
		} else if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("remote path exists but is not a directory: %s", remote)
			}
		} else {
			return fmt.Errorf("failed to stat remote directory: %w", err)
		}
	}

	// Create .lnkr.toml file if it doesn't exist
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Create new configuration using struct to maintain field order
		cfg := Config{
			Local:          currentDir,
			Remote:         remote,
			Source:         "local",
			LinkType:       LinkTypeSymbolic,
			GitExcludePath: gitExcludePath,
			Links:          []Link{},
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}
		defer file.Close()

		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode configuration: %w", err)
		}

		fmt.Printf("Created %s with local and remote directories\n", filename)
	} else {
		// Update existing configuration file
		content, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}

		var cfg Config
		if len(content) > 0 {
			if _, err := toml.Decode(string(content), &cfg); err != nil {
				return fmt.Errorf("failed to decode configuration: %w", err)
			}
		}

		// Always update local and remote
		cfg.Local = currentDir
		cfg.Remote = remote

		// Set defaults if not present
		if strings.TrimSpace(cfg.Source) == "" {
			cfg.Source = "local"
		}
		if strings.TrimSpace(cfg.LinkType) == "" {
			cfg.LinkType = LinkTypeSymbolic
		}
		if strings.TrimSpace(cfg.GitExcludePath) == "" {
			cfg.GitExcludePath = gitExcludePath
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}
		defer file.Close()

		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode configuration: %w", err)
		}

		fmt.Printf("Updated local and remote in %s\n", filename)
	}
	return nil
}

// addToGitExclude adds .lnkr.toml to .git/info/exclude
func addToGitExclude() error {
	return addToGitExcludeWithSection(ConfigFileName)
}

// addToGitExcludeWithSection adds entries to .git/info/exclude with section markers
func addToGitExcludeWithSection(entry string) error {
	return addMultipleToGitExclude([]string{entry})
}

// addMultipleToGitExclude adds multiple entries to .git/info/exclude with section markers
func addMultipleToGitExclude(entries []string) error {
	// Load config to get git exclude path
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	excludePath := config.GetGitExcludePath()
	excludeDir := filepath.Dir(excludePath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(excludeDir, 0755); err != nil {
		return err
	}

	// Read existing content
	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if section already exists
	lines := strings.Split(string(content), "\n")
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

	// Collect existing entries from the section
	existingEntries := make(map[string]struct{})
	if sectionStart != -1 && sectionEnd != -1 {
		for i := sectionStart + 1; i < sectionEnd; i++ {
			line := strings.TrimSpace(lines[i])
			if line != "" && !strings.HasPrefix(line, "#") {
				// Add / prefix if not already present
				if !strings.HasPrefix(line, "/") {
					line = "/" + line
				}
				existingEntries[line] = struct{}{}
			}
		}
	}

	// Add new entries to existing ones
	for _, entry := range entries {
		// Add / prefix if not already present
		if !strings.HasPrefix(entry, "/") {
			entry = "/" + entry
		}
		existingEntries[entry] = struct{}{}
	}

	// Convert back to slice and sort
	var allEntries []string
	for entry := range existingEntries {
		allEntries = append(allEntries, entry)
	}
	sort.Strings(allEntries)

	// Remove existing section if it exists
	if sectionStart != -1 && sectionEnd != -1 {
		lines = append(lines[:sectionStart], lines[sectionEnd+1:]...)
	}

	// Add new section at the end
	lines = append(lines, GitExcludeSectionStart)
	lines = append(lines, allEntries...)
	lines = append(lines, GitExcludeSectionEnd)

	// Write back to file
	file, err := os.Create(excludePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(strings.Join(lines, "\n"))
	if err != nil {
		return err
	}

	if len(entries) == 1 {
		fmt.Printf("Added %s to %s\n", entries[0], excludePath)
	} else {
		fmt.Printf("Added %d entries to %s\n", len(entries), excludePath)
	}
	return nil
}
