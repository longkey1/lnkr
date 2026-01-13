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
	GitExcludeSectionStart = "### LNKR STA"
	GitExcludeSectionEnd   = "### LNKR END"
)

// Link type constants
const (
	LinkTypeHard     = "hard"
	LinkTypeSymbolic = "symbolic"
)

// Default remote depth constant
const DefaultRemoteDepth = 2

type Link struct {
	Path string `toml:"path"`
	Type string `toml:"type"`
}

type Config struct {
	Local  string `toml:"local"`
	Remote string `toml:"remote"`
	// Source determines which side is treated as the source when creating links.
	// Accepts "local" or "remote". Defaults to "local" if empty or invalid.
	Source string `toml:"source"`
	// LinkType determines the default link type when adding new links.
	// Accepts "hard" or "symbolic". Defaults to "hard" if empty or invalid.
	LinkType       string `toml:"link_type"`
	GitExcludePath string `toml:"git_exclude_path"`
	Links          []Link `toml:"links"`
}

// GetSource returns normalized source value ("local" or "remote").
// Defaults to "local" when unset or invalid.
func (c *Config) GetSource() string {
	switch strings.ToLower(strings.TrimSpace(c.Source)) {
	case "remote":
		return "remote"
	default:
		return "local"
	}
}

// GetLinkType returns normalized link type value ("hard" or "symbolic").
// Defaults to "hard" when unset or invalid.
func (c *Config) GetLinkType() string {
	switch strings.ToLower(strings.TrimSpace(c.LinkType)) {
	case LinkTypeSymbolic:
		return LinkTypeSymbolic
	default:
		return LinkTypeHard
	}
}

// GetDefaultRemotePath returns the default remote path based on base directory and remote directory
func GetDefaultRemotePath(baseDir, remoteDir string, depth int) string {
	// Split the base directory path into components
	pathComponents := strings.Split(baseDir, string(os.PathSeparator))

	// Remove empty components (happens with absolute paths)
	var cleanComponents []string
	for _, component := range pathComponents {
		if component != "" {
			cleanComponents = append(cleanComponents, component)
		}
	}

	// Adjust depth if we don't have enough components
	if len(cleanComponents) < depth {
		depth = len(cleanComponents)
	}

	// Get the components for the remote path
	// depth=1: current directory only
	// depth=2: parent directory + current directory
	// depth=3: grandparent directory + parent directory + current directory
	startIndex := len(cleanComponents) - depth
	if startIndex < 0 {
		startIndex = 0
	}

	remoteComponents := cleanComponents[startIndex:]
	remotePath := strings.Join(remoteComponents, string(os.PathSeparator))
	return filepath.Join(remoteDir, remotePath)
}

func loadConfig() (*Config, error) {
	filename := ConfigFileName
	config := &Config{}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return config, nil
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if len(content) > 0 {
		if _, err := toml.Decode(string(content), config); err != nil {
			return nil, err
		}
	}

	if err := validateSource(config.Source); err != nil {
		return nil, err
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

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return err
	}

	return nil
}

// GetGitExcludePath returns the git exclude path from config or default value
func (c *Config) GetGitExcludePath() string {
	if c.GitExcludePath != "" {
		return c.GitExcludePath
	}
	return GitExcludePath
}

func validateSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return nil
	}

	switch normalized := strings.ToLower(strings.TrimSpace(source)); normalized {
	case "local", "remote":
		return nil
	default:
		return fmt.Errorf("invalid source value %q in %s: expected \"local\" or \"remote\"", source, ConfigFileName)
	}
}

func validateLinkType(linkType string) error {
	if strings.TrimSpace(linkType) == "" {
		return nil
	}

	switch normalized := strings.ToLower(strings.TrimSpace(linkType)); normalized {
	case LinkTypeHard, LinkTypeSymbolic:
		return nil
	default:
		return fmt.Errorf("invalid link_type value %q in %s: expected \"hard\" or \"symbolic\"", linkType, ConfigFileName)
	}
}
