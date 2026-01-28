package lnkr

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config keys
const (
	ConfigKeyRemoteRoot     = "remote_root"
	ConfigKeyLocalRoot      = "local_root"
	ConfigKeyLinkType       = "link_type"
	ConfigKeyGitExcludePath = "git_exclude_path"
)

// InitGlobalConfig initializes viper with global configuration settings.
// This should be called once at application startup.
func InitGlobalConfig() {
	// Set config file location
	homeDir, err := os.UserHomeDir()
	if err == nil {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(filepath.Join(homeDir, ".config", "lnkr"))
	}

	// Set default values
	if homeDir != "" {
		viper.SetDefault(ConfigKeyRemoteRoot, filepath.Join(homeDir, ".config", "lnkr"))
	}
	// local_root has no default - when empty, uses current directory name only
	viper.SetDefault(ConfigKeyLinkType, LinkTypeSymbolic)
	viper.SetDefault(ConfigKeyGitExcludePath, GitExcludePath)

	// Enable environment variable binding
	// LNKR_REMOTE_ROOT -> remote_root
	viper.SetEnvPrefix("LNKR")
	viper.AutomaticEnv()

	// Explicitly bind environment variables to config keys
	// This ensures LNKR_LOCAL_ROOT maps to "local_root" key correctly
	_ = viper.BindEnv(ConfigKeyRemoteRoot, "LNKR_REMOTE_ROOT")
	_ = viper.BindEnv(ConfigKeyLocalRoot, "LNKR_LOCAL_ROOT")
	_ = viper.BindEnv(ConfigKeyLinkType, "LNKR_LINK_TYPE")
	_ = viper.BindEnv(ConfigKeyGitExcludePath, "LNKR_GIT_EXCLUDE_PATH")

	// Read config file (ignore error if not found)
	_ = viper.ReadInConfig()
}

// GetRemoteRoot returns the remote root directory with environment variables expanded.
// Priority: environment variable > config file > default value
func GetRemoteRoot() string {
	return os.ExpandEnv(viper.GetString(ConfigKeyRemoteRoot))
}

// GetLocalRoot returns the local root directory for calculating relative paths with environment variables expanded.
// Priority: environment variable > config file > empty (uses current dir name only)
func GetLocalRoot() string {
	return os.ExpandEnv(viper.GetString(ConfigKeyLocalRoot))
}

// GetGlobalLinkType returns the default link type.
// Priority: environment variable > config file > default value
func GetGlobalLinkType() string {
	return viper.GetString(ConfigKeyLinkType)
}

// GetGlobalGitExcludePath returns the default git exclude path.
// Priority: environment variable > config file > default value
func GetGlobalGitExcludePath() string {
	return viper.GetString(ConfigKeyGitExcludePath)
}
