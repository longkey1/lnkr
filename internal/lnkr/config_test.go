package lnkr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestLoadConfigLinkTypeValidation(t *testing.T) {
	// Cannot use t.Parallel() with os.Chdir()
	testCases := []struct {
		name         string
		linkType     string
		wantErr      bool
		wantLinkType string
	}{
		{
			name:         "Hard",
			linkType:     "hard",
			wantLinkType: "hard",
		},
		{
			name:         "SymbolicUppercase",
			linkType:     "SYMBOLIC",
			wantLinkType: "sym",
		},
		{
			name:         "Sym",
			linkType:     "sym",
			wantLinkType: "sym",
		},
		{
			name:         "Empty",
			linkType:     "",
			wantLinkType: "sym",
		},
		{
			name:     "Invalid",
			linkType: "softlink",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			originalWD, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get working directory: %v", err)
			}
			t.Cleanup(func() {
				if chdirErr := os.Chdir(originalWD); chdirErr != nil {
					t.Fatalf("failed to restore working directory: %v", chdirErr)
				}
			})

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}

			configContent := fmt.Sprintf(`local = "/tmp/local"
remote = "/tmp/remote"
link_type = "%s"
`, tc.linkType)

			configPath := filepath.Join(tempDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := loadConfig()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := cfg.GetLinkType()
			if got != tc.wantLinkType {
				t.Fatalf("unexpected link_type: got %q, want %q", got, tc.wantLinkType)
			}
		})
	}
}

func TestGetDefaultRemotePath(t *testing.T) {
	testCases := []struct {
		name       string
		currentDir string
		localRoot  string
		remoteRoot string
		want       string
	}{
		{
			name:       "UnderLocalRootUsesRelativePath",
			currentDir: "/home/user/src/github.com/owner/project",
			localRoot:  "/home/user/src",
			remoteRoot: "/backup",
			want:       "/backup/github.com/owner/project",
		},
		{
			name:       "EqualToLocalRootUsesRemoteRoot",
			currentDir: "/home/user/src",
			localRoot:  "/home/user/src",
			remoteRoot: "/backup",
			want:       "/backup",
		},
		{
			name:       "OutsideLocalRootFallsBackToBasename",
			currentDir: "/opt/project",
			localRoot:  "/home/user/src",
			remoteRoot: "/backup",
			want:       "/backup/project",
		},
		{
			name:       "EmptyLocalRootUsesBasename",
			currentDir: "/home/user/src/project",
			localRoot:  "",
			remoteRoot: "/backup",
			want:       "/backup/project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetDefaultRemotePath(tc.currentDir, tc.localRoot, tc.remoteRoot)
			if got != tc.want {
				t.Fatalf("GetDefaultRemotePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetGitExcludePath(t *testing.T) {
	testCases := []struct {
		name           string
		gitExcludePath string
		want           string
	}{
		{
			name:           "CustomPath",
			gitExcludePath: ".git/info/custom",
			want:           ".git/info/custom",
		},
		{
			name:           "EmptyUsesDefault",
			gitExcludePath: "",
			want:           GitExcludePath,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{GitExcludePath: tc.gitExcludePath}
			if got := cfg.GetGitExcludePath(); got != tc.want {
				t.Fatalf("GetGitExcludePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSaveAndLoadConfigRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	want := &Config{
		Remote:         "/tmp/remote",
		LinkType:       LinkTypeHard,
		GitExcludePath: ".git/info/custom",
		Links: []Link{
			{Path: "a.txt", Type: LinkTypeSymbolic},
			{Path: "conf/b.txt", Type: LinkTypeHard},
		},
	}

	if err := saveConfig(want); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	got, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if got.Remote != want.Remote ||
		got.LinkType != want.LinkType || got.GitExcludePath != want.GitExcludePath {
		t.Fatalf("unexpected config: got %+v, want %+v", got, want)
	}
	if len(got.Links) != len(want.Links) {
		t.Fatalf("unexpected number of links: got %d, want %d", len(got.Links), len(want.Links))
	}
	for i, link := range want.Links {
		if got.Links[i] != link {
			t.Fatalf("unexpected link at %d: got %+v, want %+v", i, got.Links[i], link)
		}
	}
}

func TestLoadConfigMissingFileReturnsError(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := loadConfig()
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

// TestLoadConfigIgnoresDeprecatedLocalKey verifies that old configuration
// files carrying a "local" key still load, and that the key is dropped on
// the next save.
func TestLoadConfigIgnoresDeprecatedLocalKey(t *testing.T) {
	t.Chdir(t.TempDir())

	content := "local = \"/somewhere\"\nremote = \"/tmp/remote\"\n"
	if err := os.WriteFile(ConfigFileName, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if config.Remote != "/tmp/remote" {
		t.Fatalf("unexpected remote: %q", config.Remote)
	}

	if err := saveConfig(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	var raw map[string]any
	if _, err := toml.DecodeFile(ConfigFileName, &raw); err != nil {
		t.Fatalf("failed to decode saved config: %v", err)
	}
	if _, ok := raw["local"]; ok {
		t.Fatalf("saved config still contains deprecated local key: %v", raw)
	}
}

func TestGitExcludeWorktreeSkip(t *testing.T) {
	testCases := []struct {
		name           string
		gitExcludePath string // empty means default
		gitIsFile      bool
		gitIsDir       bool
		wantSkip       bool
	}{
		{
			name:      "DefaultPathWithGitFile",
			gitIsFile: true,
			wantSkip:  true,
		},
		{
			name:     "DefaultPathWithGitDir",
			gitIsDir: true,
			wantSkip: false,
		},
		{
			name:     "DefaultPathWithoutGit",
			wantSkip: false,
		},
		{
			name:           "CustomPathUnderGitWithGitFile",
			gitExcludePath: ".git/info/custom",
			gitIsFile:      true,
			wantSkip:       true,
		},
		{
			name:           "RelativePathOutsideGit",
			gitExcludePath: "exclude.txt",
			gitIsFile:      true,
			wantSkip:       false,
		},
		{
			name:           "AbsolutePath",
			gitExcludePath: "/tmp/exclude",
			gitIsFile:      true,
			wantSkip:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			gitPath := filepath.Join(dir, ".git")
			if tc.gitIsFile {
				if err := os.WriteFile(gitPath, []byte("gitdir: /elsewhere\n"), 0644); err != nil {
					t.Fatalf("failed to write .git file: %v", err)
				}
			}
			if tc.gitIsDir {
				if err := os.MkdirAll(gitPath, 0755); err != nil {
					t.Fatalf("failed to create .git dir: %v", err)
				}
			}

			cfg := &Config{GitExcludePath: tc.gitExcludePath, dir: dir}
			gotPath, gotSkip := cfg.gitExcludeWorktreeSkip()
			if gotSkip != tc.wantSkip {
				t.Fatalf("gitExcludeWorktreeSkip() skip = %v, want %v", gotSkip, tc.wantSkip)
			}
			if gotSkip && gotPath != gitPath {
				t.Fatalf("gitExcludeWorktreeSkip() path = %q, want %q", gotPath, gitPath)
			}
		})
	}
}
