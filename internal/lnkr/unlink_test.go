package lnkr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnlink(t *testing.T) {
	testCases := []struct {
		name        string
		remoteFiles map[string]string
		links       []Link
	}{
		{
			name:        "SymbolicFile",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:        "HardFile",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeHard}},
		},
		{
			name:        "HardDirectory",
			remoteFiles: map[string]string{"conf/a.txt": "a", "conf/b.txt": "b"},
			links:       []Link{{Path: "conf", Type: LinkTypeHard}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localDir, remoteDir := setupProject(t, &Config{Links: tc.links})
			writeFiles(t, remoteDir, tc.remoteFiles)

			if err := CreateLinks(); err != nil {
				t.Fatalf("failed to create links: %v", err)
			}

			if err := Unlink(false, true); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Local links must be removed, remote files must be intact.
			for _, link := range tc.links {
				if _, err := os.Lstat(filepath.Join(localDir, link.Path)); !os.IsNotExist(err) {
					t.Fatalf("local link still exists: %s", link.Path)
				}
			}
			for path := range tc.remoteFiles {
				if _, err := os.Stat(filepath.Join(remoteDir, path)); err != nil {
					t.Fatalf("remote file missing after unlink: %v", err)
				}
			}

			// The LNKR section must be removed from the git exclude file.
			content, err := os.ReadFile(GitExcludePath)
			if err != nil {
				t.Fatalf("failed to read exclude file: %v", err)
			}
			if strings.Contains(string(content), GitExcludeSectionStart) {
				t.Fatalf("LNKR section still present in exclude file:\n%s", content)
			}
		})
	}
}

func TestUnlinkKeepsUnlinkedFilesInHardDir(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{
		Links: []Link{{Path: "conf", Type: LinkTypeHard}},
	})
	writeFiles(t, remoteDir, map[string]string{"conf/a.txt": "a"})

	if err := CreateLinks(); err != nil {
		t.Fatalf("failed to create links: %v", err)
	}

	// A file added after linking is not a hard link to remote.
	writeFiles(t, localDir, map[string]string{"conf/extra.txt": "extra"})

	if err := Unlink(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The linked file must be removed, the unrelated file must survive.
	if _, err := os.Lstat(filepath.Join(localDir, "conf", "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("linked file still exists after unlink")
	}
	content, err := os.ReadFile(filepath.Join(localDir, "conf", "extra.txt"))
	if err != nil {
		t.Fatalf("unrelated file was removed by unlink: %v", err)
	}
	if string(content) != "extra" {
		t.Fatalf("unexpected content: got %q, want %q", content, "extra")
	}
}

func TestUnlinkDryRun(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{
		Links: []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
	})
	writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})

	if err := CreateLinks(); err != nil {
		t.Fatalf("failed to create links: %v", err)
	}

	if err := Unlink(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The link must still be in place after a dry run.
	assertLink(t, filepath.Join(localDir, "a.txt"), filepath.Join(remoteDir, "a.txt"), LinkTypeSymbolic)
}

func TestUnlinkMissingLocalSkipped(t *testing.T) {
	setupProject(t, &Config{
		Links: []Link{{Path: "ghost.txt", Type: LinkTypeSymbolic}},
	})

	if err := Unlink(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnlinkNoLinks(t *testing.T) {
	setupProject(t, &Config{Links: []Link{}})

	if err := Unlink(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveAllLinksFromGitExclude(t *testing.T) {
	testCases := []struct {
		name            string
		existingContent string // empty means the exclude file does not exist
		wantContent     string
	}{
		{
			name: "RemovesSectionKeepsOtherEntries",
			existingContent: "node_modules\n" +
				GitExcludeSectionStart + "\n/.lnkr.toml\n/a.txt\n" + GitExcludeSectionEnd + "\nvendor",
			wantContent: "node_modules\nvendor",
		},
		{
			name:            "NoSectionLeavesContentUnchanged",
			existingContent: "node_modules\nvendor\n",
			wantContent:     "node_modules\nvendor\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			excludePath := filepath.Join(tempDir, "exclude")
			if err := os.WriteFile(excludePath, []byte(tc.existingContent), 0644); err != nil {
				t.Fatalf("failed to write exclude file: %v", err)
			}

			config := &Config{
				GitExcludePath: excludePath,
				Links:          []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			}
			if err := removeAllLinksFromGitExclude(config); err != nil {
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

func TestRemoveAllLinksFromGitExcludeMissingFile(t *testing.T) {
	config := &Config{
		GitExcludePath: filepath.Join(t.TempDir(), "missing"),
		Links:          []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
	}
	if err := removeAllLinksFromGitExclude(config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
