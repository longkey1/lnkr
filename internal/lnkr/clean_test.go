package lnkr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClean(t *testing.T) {
	testCases := []struct {
		name           string
		createConfig   bool
		excludeContent string // empty means the exclude file is not created
		wantExclude    string
	}{
		{
			name:           "RemovesConfigAndExcludeEntry",
			createConfig:   true,
			excludeContent: "node_modules\n" + ConfigFileName + "\nvendor",
			wantExclude:    "node_modules\nvendor",
		},
		{
			name:           "NoConfigFile",
			excludeContent: ConfigFileName + "\n",
			wantExclude:    "",
		},
		{
			name:         "NoExcludeFile",
			createConfig: true,
		},
		{
			name:         "RemovesLnkrSection",
			createConfig: true,
			excludeContent: "node_modules\n" +
				GitExcludeSectionStart + "\n/.lnkr.toml\n/a.txt\n" + GitExcludeSectionEnd + "\nvendor",
			wantExclude: "node_modules\nvendor",
		},
		{
			name:         "RemovesLegacyMarkerSection",
			createConfig: true,
			excludeContent: "node_modules\n" +
				legacyGitExcludeSectionStart + "\n/.lnkr.toml\n" + GitExcludeSectionEnd + "\nvendor",
			wantExclude: "node_modules\nvendor",
		},
		{
			name:           "EntryNotInExclude",
			createConfig:   true,
			excludeContent: "node_modules\n",
			wantExclude:    "node_modules\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			if tc.createConfig {
				if err := saveConfig(&Config{Local: "/tmp/local", Remote: "/tmp/remote"}); err != nil {
					t.Fatalf("failed to save config: %v", err)
				}
			}

			if tc.excludeContent != "" {
				if err := os.MkdirAll(filepath.Dir(GitExcludePath), 0755); err != nil {
					t.Fatalf("failed to create exclude dir: %v", err)
				}
				if err := os.WriteFile(GitExcludePath, []byte(tc.excludeContent), 0644); err != nil {
					t.Fatalf("failed to write exclude file: %v", err)
				}
			}

			if err := Clean(false, true); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The configuration file must be gone.
			if _, err := os.Stat(ConfigFileName); !os.IsNotExist(err) {
				t.Fatalf("%s still exists after clean", ConfigFileName)
			}

			if tc.excludeContent == "" {
				return
			}

			content, err := os.ReadFile(GitExcludePath)
			if err != nil {
				t.Fatalf("failed to read exclude file: %v", err)
			}
			if string(content) != tc.wantExclude {
				t.Fatalf("unexpected exclude content: got %q, want %q", content, tc.wantExclude)
			}
		})
	}
}

func TestRemoveFromGitExcludeWithPath(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		entry       string
		wantContent string
	}{
		{
			name:        "RemovesMatchingLine",
			content:     "a\n.lnkr.toml\nb",
			entry:       ".lnkr.toml",
			wantContent: "a\nb",
		},
		{
			name:        "RemovesLineWithSurroundingSpaces",
			content:     "a\n  .lnkr.toml  \nb",
			entry:       ".lnkr.toml",
			wantContent: "a\nb",
		},
		{
			name:        "EntryNotFoundLeavesContentUnchanged",
			content:     "a\nb\n",
			entry:       ".lnkr.toml",
			wantContent: "a\nb\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			excludePath := filepath.Join(t.TempDir(), "exclude")
			if err := os.WriteFile(excludePath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write exclude file: %v", err)
			}

			if err := removeFromGitExcludeWithPath(excludePath, tc.entry); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, err := os.ReadFile(excludePath)
			if err != nil {
				t.Fatalf("failed to read exclude file: %v", err)
			}
			if string(content) != tc.wantContent {
				t.Fatalf("unexpected content: got %q, want %q", content, tc.wantContent)
			}
		})
	}
}

func TestRemoveFromGitExcludeWithPathMissingFile(t *testing.T) {
	excludePath := filepath.Join(t.TempDir(), "missing")
	if err := removeFromGitExcludeWithPath(excludePath, ConfigFileName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
