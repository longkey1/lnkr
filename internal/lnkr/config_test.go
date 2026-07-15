package lnkr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
		Local:          "/tmp/local",
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

	if got.Local != want.Local || got.Remote != want.Remote ||
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
