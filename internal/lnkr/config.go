package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Configuration file name constant
const ConfigFileName = ".lnkr.toml"

// Git exclude file path constant
const GitExcludePath = ".git/info/exclude"

// Git exclude section markers
const (
	GitExcludeSectionStart = "### LNKR START"
	GitExcludeSectionEnd   = "### LNKR END"

	// legacyGitExcludeSectionStart is the start marker written by older
	// versions. It is still recognized when reading exclude files.
	legacyGitExcludeSectionStart = "### LNKR STA"
)

// ErrConfigNotFound is returned when the project configuration file does not
// exist in the current directory.
var ErrConfigNotFound = fmt.Errorf("%s not found in current or any parent directory, run 'lnkr init' first", ConfigFileName)

// Link type constants
const (
	LinkTypeHard     = "hard"
	LinkTypeSymbolic = "sym"
)

type Link struct {
	Path string `toml:"path"`
	Type string `toml:"type"`
}

type Config struct {
	Local  string `toml:"local"`
	Remote string `toml:"remote"`
	// LinkType determines the default link type when adding new links.
	// Accepts "hard" or "sym" ("symbolic" is accepted as an alias).
	// Defaults to "sym" if empty or invalid.
	LinkType       string `toml:"link_type"`
	GitExcludePath string `toml:"git_exclude_path"`
	Links          []Link `toml:"links"`

	// dir is the absolute path of the directory containing the loaded
	// configuration file. Empty for configs not loaded from disk; relative
	// paths then resolve against the current directory as before.
	dir string
}

// GetLinkType returns normalized link type value ("hard" or "sym").
// Defaults to "sym" when unset or invalid.
// Accepts "symbolic" as an alias for "sym" for backward compatibility.
func (c *Config) GetLinkType() string {
	switch strings.ToLower(strings.TrimSpace(c.LinkType)) {
	case LinkTypeHard:
		return LinkTypeHard
	case LinkTypeSymbolic, "symbolic":
		return LinkTypeSymbolic
	default:
		return LinkTypeSymbolic
	}
}

// GetDefaultRemotePath returns the default remote path based on current directory, local root, and remote root.
// If localRoot is provided, calculates relative path from localRoot.
// If localRoot is empty, uses only the current directory name.
func GetDefaultRemotePath(currentDir, localRoot, remoteRoot string) string {
	var relativePath string

	if localRoot != "" {
		// Calculate relative path from localRoot
		rel, err := filepath.Rel(localRoot, currentDir)
		if err == nil && !strings.HasPrefix(rel, "..") {
			relativePath = rel
		} else {
			// If currentDir is not under localRoot, fall back to basename
			relativePath = filepath.Base(currentDir)
		}
	} else {
		// No localRoot set, use only the current directory name
		relativePath = filepath.Base(currentDir)
	}

	return filepath.Join(remoteRoot, relativePath)
}

// findConfigFile locates the configuration file by walking up from the
// current directory (like git does), so commands work from subdirectories.
func findConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	for {
		candidate := filepath.Join(dir, ConfigFileName)
		if _, err := os.Lstat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrConfigNotFound
		}
		dir = parent
	}
}

func loadConfig() (*Config, error) {
	filename, err := findConfigFile()
	if err != nil {
		return nil, err
	}
	config := &Config{dir: filepath.Dir(filename)}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if len(content) > 0 {
		if _, err := toml.Decode(string(content), config); err != nil {
			return nil, err
		}
	}

	if err := validateLinkType(config.LinkType); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadConfigForCLI loads the configuration file for CLI commands.
// This is an exported wrapper around loadConfig for use in cmd package.
func LoadConfigForCLI() (*Config, error) {
	return loadConfig()
}

func saveConfig(config *Config) error {
	filename := ConfigFileName
	if config.dir != "" {
		filename = filepath.Join(config.dir, ConfigFileName)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		_ = file.Close()
		return err
	}

	return file.Close()
}

// findGitExcludeSection returns the line indexes of the LNKR section start and
// end markers, or (-1, -1) when the section does not exist. Both the current
// and the legacy start marker are recognized.
func findGitExcludeSection(lines []string) (int, int) {
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start == -1 && (trimmed == GitExcludeSectionStart || trimmed == legacyGitExcludeSectionStart) {
			start = i
			continue
		}
		if start != -1 && trimmed == GitExcludeSectionEnd {
			return start, i
		}
	}
	return -1, -1
}

// GetGitExcludePath returns the git exclude path from config or default value.
// A relative path is anchored at the directory containing the configuration
// file, so commands work from subdirectories.
func (c *Config) GetGitExcludePath() string {
	path := c.GitExcludePath
	if path == "" {
		path = GitExcludePath
	}
	if !filepath.IsAbs(path) && c.dir != "" {
		return filepath.Join(c.dir, path)
	}
	return path
}

// GetLocalExpanded returns the expanded local path with environment variables resolved.
// Returns error if any variable in the path is undefined.
func (c *Config) GetLocalExpanded() (string, error) {
	return ExpandPath(c.Local)
}

// GetRemoteExpanded returns the expanded remote path with environment variables resolved.
// Returns error if any variable in the path is undefined.
func (c *Config) GetRemoteExpanded() (string, error) {
	return ExpandPath(c.Remote)
}

func validateLinkType(linkType string) error {
	if strings.TrimSpace(linkType) == "" {
		return nil
	}

	switch normalized := strings.ToLower(strings.TrimSpace(linkType)); normalized {
	case LinkTypeHard, LinkTypeSymbolic, "symbolic":
		return nil
	default:
		return fmt.Errorf("invalid link_type value %q in %s: expected \"hard\" or \"sym\"", linkType, ConfigFileName)
	}
}
