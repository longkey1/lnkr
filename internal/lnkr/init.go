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
func Init(remote string, gitExcludePath string, force bool) error {
	if err := createLnkTomlWithRemote(remote, gitExcludePath, force); err != nil {
		return fmt.Errorf("failed to create %s: %w", ConfigFileName, err)
	}

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Move .lnkr.toml to remote and create symbolic link
	if config.Remote != "" {
		if err := setupConfigSymlink(config); err != nil {
			return fmt.Errorf("failed to setup %s symlink: %w", ConfigFileName, err)
		}
	}

	if err := applyAllLinksToGitExclude(config); err != nil {
		return fmt.Errorf("failed to add to %s: %w", GitExcludePath, err)
	}

	fmt.Println("Project initialized successfully!")
	return nil
}

// setupConfigSymlink moves .lnkr.toml to remote and creates a symbolic link
func setupConfigSymlink(config *Config) error {
	localDir, err := config.GetLocalExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand local path: %w", err)
	}
	remoteDir, err := config.GetRemoteExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand remote path: %w", err)
	}
	localPath := filepath.Join(localDir, ConfigFileName)
	remotePath := filepath.Join(remoteDir, ConfigFileName)

	// Check if local path is already a symlink pointing to correct location
	if fi, err := os.Lstat(localPath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(localPath); err == nil && target == remotePath {
			return nil // Already correctly configured
		}
		_ = os.Remove(localPath)
	}

	// If remote already exists, just create symlink
	if _, err := os.Stat(remotePath); err == nil {
		_ = os.Remove(localPath)
	} else if os.IsNotExist(err) {
		// Move local to remote
		if err := os.Rename(localPath, remotePath); err != nil {
			return fmt.Errorf("failed to move %s to remote: %w", ConfigFileName, err)
		}
		fmt.Printf("Moved: %s -> %s\n", localPath, remotePath)
	} else {
		return fmt.Errorf("failed to stat remote %s: %w", ConfigFileName, err)
	}

	// Create symbolic link using shared function
	return createLink(remotePath, localPath, LinkTypeSymbolic)
}

// createLnkTomlWithRemote creates the .lnkr.toml file with remote if it doesn't exist
func createLnkTomlWithRemote(remote string, gitExcludePath string, force bool) error {
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

	// If the config file is a symlink pointing at a different remote,
	// replace it with a regular file so the new settings are written
	// locally; setupConfigSymlink then moves it to the new remote.
	if fi, err := os.Lstat(filename); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		target, readErr := os.Readlink(filename)
		if readErr != nil || (remote != "" && target != filepath.Join(remote, ConfigFileName)) {
			content, contentErr := os.ReadFile(filename)
			if contentErr != nil && !os.IsNotExist(contentErr) {
				return fmt.Errorf("failed to read %s: %w", filename, contentErr)
			}
			if err := os.Remove(filename); err != nil {
				return fmt.Errorf("failed to replace %s symlink: %w", filename, err)
			}
			if contentErr == nil {
				if err := os.WriteFile(filename, content, 0644); err != nil {
					return fmt.Errorf("failed to rewrite %s: %w", filename, err)
				}
			}
		}
	}

	// Create .lnkr.toml file if it doesn't exist
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Create new configuration using struct to maintain field order
		// Use ContractPath to make paths portable
		cfg := Config{
			Local:          ContractPath(currentDir),
			Remote:         ContractPath(remote),
			LinkType:       GetGlobalLinkType(),
			GitExcludePath: gitExcludePath,
			Links:          []Link{},
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}

		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(cfg); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to encode configuration: %w", err)
		}

		// Add commented link entry for .lnkr.toml
		configLinkComment := "\n# .lnkr.toml is automatically managed as a symbolic link to remote\n# [[links]]\n# path = \".lnkr.toml\"\n# type = \"sym\"\n"
		if _, err := file.WriteString(configLinkComment); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to write config link comment: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close configuration file: %w", err)
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

		// Refuse to change existing local/remote settings unless forced,
		// so a stray re-init cannot silently break existing links.
		newLocal := ContractPath(currentDir)
		newRemote := ContractPath(remote)
		if !force {
			if (cfg.Local != "" && cfg.Local != newLocal) || (cfg.Remote != "" && cfg.Remote != newRemote) {
				return fmt.Errorf("%s already exists with different settings (local: %q, remote: %q); use --force to overwrite", filename, cfg.Local, cfg.Remote)
			}
		}
		cfg.Local = newLocal
		cfg.Remote = newRemote

		// Set defaults if not present
		if strings.TrimSpace(cfg.LinkType) == "" {
			cfg.LinkType = GetGlobalLinkType()
		}
		if strings.TrimSpace(cfg.GitExcludePath) == "" {
			cfg.GitExcludePath = gitExcludePath
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}

		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(cfg); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to encode configuration: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close configuration file: %w", err)
		}

		fmt.Printf("Updated local and remote in %s\n", filename)
	}
	return nil
}

// addMultipleToGitExclude adds multiple entries to .git/info/exclude with section markers
func addMultipleToGitExclude(config *Config, entries []string) error {
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
	sectionStart, sectionEnd := findGitExcludeSection(lines)

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
	if err := os.WriteFile(excludePath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}

	if len(entries) == 1 {
		fmt.Printf("Added %s to %s\n", entries[0], excludePath)
	} else {
		fmt.Printf("Added %d entries to %s\n", len(entries), excludePath)
	}
	return nil
}
