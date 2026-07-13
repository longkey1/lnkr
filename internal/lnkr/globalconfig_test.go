package lnkr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetGlobalConfig isolates viper state for a test and restores a clean
// state afterwards so other tests are not affected.
func resetGlobalConfig(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	// Point HOME at an empty directory so the user's real global config
	// file is never read, and clear LNKR variables from the environment.
	t.Setenv("HOME", t.TempDir())
	for _, key := range []string{"LNKR_REMOTE_ROOT", "LNKR_LOCAL_ROOT", "LNKR_LINK_TYPE", "LNKR_GIT_EXCLUDE_PATH"} {
		t.Setenv(key, "")
	}
}

func TestGlobalConfigDefaults(t *testing.T) {
	resetGlobalConfig(t)

	InitGlobalConfig()

	home := os.Getenv("HOME")
	if got, want := GetRemoteRoot(), filepath.Join(home, ".config", "lnkr"); got != want {
		t.Fatalf("unexpected remote root: got %q, want %q", got, want)
	}
	if got := GetLocalRoot(); got != "" {
		t.Fatalf("unexpected local root: got %q, want empty", got)
	}
	if got := GetGlobalLinkType(); got != LinkTypeSymbolic {
		t.Fatalf("unexpected link type: got %q, want %q", got, LinkTypeSymbolic)
	}
	if got := GetGlobalGitExcludePath(); got != GitExcludePath {
		t.Fatalf("unexpected git exclude path: got %q, want %q", got, GitExcludePath)
	}
}

func TestGlobalConfigFromFile(t *testing.T) {
	resetGlobalConfig(t)

	home := os.Getenv("HOME")
	configDir := filepath.Join(home, ".config", "lnkr")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `remote_root = "$HOME/backup"
local_root = "/cfg/local"
link_type = "hard"
git_exclude_path = ".git/info/custom"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	InitGlobalConfig()

	if got, want := GetRemoteRoot(), filepath.Join(home, "backup"); got != want {
		t.Fatalf("unexpected remote root: got %q, want %q", got, want)
	}
	if got := GetLocalRoot(); got != "/cfg/local" {
		t.Fatalf("unexpected local root: got %q, want %q", got, "/cfg/local")
	}
	if got := GetGlobalLinkType(); got != LinkTypeHard {
		t.Fatalf("unexpected link type: got %q, want %q", got, LinkTypeHard)
	}
	if got := GetGlobalGitExcludePath(); got != ".git/info/custom" {
		t.Fatalf("unexpected git exclude path: got %q, want %q", got, ".git/info/custom")
	}
}

func TestGlobalConfigEnvOverridesFile(t *testing.T) {
	resetGlobalConfig(t)

	home := os.Getenv("HOME")
	configDir := filepath.Join(home, ".config", "lnkr")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `remote_root = "/from/file"
local_root = "/from/file/local"
link_type = "hard"
git_exclude_path = ".git/info/file"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("LNKR_REMOTE_ROOT", "/from/env")
	t.Setenv("LNKR_LOCAL_ROOT", "/from/env/local")
	t.Setenv("LNKR_LINK_TYPE", LinkTypeSymbolic)
	t.Setenv("LNKR_GIT_EXCLUDE_PATH", ".git/info/env")

	InitGlobalConfig()

	if got := GetRemoteRoot(); got != "/from/env" {
		t.Fatalf("unexpected remote root: got %q, want %q", got, "/from/env")
	}
	if got := GetLocalRoot(); got != "/from/env/local" {
		t.Fatalf("unexpected local root: got %q, want %q", got, "/from/env/local")
	}
	if got := GetGlobalLinkType(); got != LinkTypeSymbolic {
		t.Fatalf("unexpected link type: got %q, want %q", got, LinkTypeSymbolic)
	}
	if got := GetGlobalGitExcludePath(); got != ".git/info/env" {
		t.Fatalf("unexpected git exclude path: got %q, want %q", got, ".git/info/env")
	}
}
