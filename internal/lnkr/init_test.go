package lnkr

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestInitWithRemote(t *testing.T) {
	tempDir := t.TempDir()
	remoteDir := filepath.Join(tempDir, "remote")
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	t.Chdir(projectDir)

	// Use a real git repository so .git/info/exclude handling is exercised
	// against the layout produced by git itself.
	gitInit := exec.Command("git", "init", "--quiet")
	gitInit.Dir = projectDir
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	if err := Init(remoteDir, GitExcludePath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// .lnkr.toml must be a symlink pointing to the copy moved to remote.
	fi, err := os.Lstat(ConfigFileName)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", ConfigFileName, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", ConfigFileName)
	}
	target, err := os.Readlink(ConfigFileName)
	if err != nil {
		t.Fatalf("failed to read link: %v", err)
	}
	wantTarget := filepath.Join(remoteDir, ConfigFileName)
	if target != wantTarget {
		t.Fatalf("unexpected symlink target: got %q, want %q", target, wantTarget)
	}
	if _, err := os.Stat(wantTarget); err != nil {
		t.Fatalf("remote config file missing: %v", err)
	}

	// The stored paths must expand back to the project and remote directories.
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	localExpanded, err := config.GetLocalExpanded()
	if err != nil {
		t.Fatalf("failed to expand local path: %v", err)
	}
	if localExpanded != cwd {
		t.Fatalf("unexpected local path: got %q, want %q", localExpanded, cwd)
	}
	remoteExpanded, err := config.GetRemoteExpanded()
	if err != nil {
		t.Fatalf("failed to expand remote path: %v", err)
	}
	if remoteExpanded != remoteDir {
		t.Fatalf("unexpected remote path: got %q, want %q", remoteExpanded, remoteDir)
	}

	// The git exclude file must contain the LNKR section with the config file.
	entries := gitExcludeSectionEntries(t, GitExcludePath)
	if !slices.Contains(entries, "/"+ConfigFileName) {
		t.Fatalf("git exclude does not contain /%s: %v", ConfigFileName, entries)
	}

	// A second init must be idempotent and keep the symlink in place.
	if err := Init(remoteDir, GitExcludePath); err != nil {
		t.Fatalf("unexpected error on re-init: %v", err)
	}
	fi, err = os.Lstat(ConfigFileName)
	if err != nil {
		t.Fatalf("failed to stat %s after re-init: %v", ConfigFileName, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to remain a symlink after re-init", ConfigFileName)
	}
}

func TestInitWithoutRemote(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Init("", GitExcludePath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without a remote, the config file must remain a regular local file.
	fi, err := os.Lstat(ConfigFileName)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", ConfigFileName, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to be a regular file, got symlink", ConfigFileName)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if config.Remote != "" {
		t.Fatalf("expected empty remote, got %q", config.Remote)
	}

	entries := gitExcludeSectionEntries(t, GitExcludePath)
	if !slices.Contains(entries, "/"+ConfigFileName) {
		t.Fatalf("git exclude does not contain /%s: %v", ConfigFileName, entries)
	}
}

func TestInitRemotePathIsFile(t *testing.T) {
	tempDir := t.TempDir()
	remotePath := filepath.Join(tempDir, "remote")
	if err := os.WriteFile(remotePath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("failed to write remote file: %v", err)
	}

	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	t.Chdir(projectDir)

	if err := Init(remotePath, GitExcludePath); err == nil {
		t.Fatalf("expected error when remote path is a file, but got none")
	}
}

func TestAddMultipleToGitExclude(t *testing.T) {
	testCases := []struct {
		name            string
		existingContent string // empty means the exclude file does not exist
		entries         []string
		wantEntries     []string
		wantContains    []string // raw content that must be preserved
	}{
		{
			name:        "CreatesNewSectionWithSortedSlashPrefixedEntries",
			entries:     []string{ConfigFileName, "b.txt", "a.txt"},
			wantEntries: []string{"/" + ConfigFileName, "/a.txt", "/b.txt"},
		},
		{
			name: "MergesWithExistingSection",
			existingContent: "node_modules\n" +
				GitExcludeSectionStart + "\n/old.txt\n" + GitExcludeSectionEnd + "\n",
			entries:      []string{"new.txt"},
			wantEntries:  []string{"/new.txt", "/old.txt"},
			wantContains: []string{"node_modules"},
		},
		{
			name:            "DeduplicatesEntries",
			existingContent: GitExcludeSectionStart + "\n/a.txt\n" + GitExcludeSectionEnd,
			entries:         []string{"a.txt", "/a.txt"},
			wantEntries:     []string{"/a.txt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			if tc.existingContent != "" {
				if err := os.MkdirAll(filepath.Dir(GitExcludePath), 0755); err != nil {
					t.Fatalf("failed to create exclude dir: %v", err)
				}
				if err := os.WriteFile(GitExcludePath, []byte(tc.existingContent), 0644); err != nil {
					t.Fatalf("failed to write exclude file: %v", err)
				}
			}

			if err := addMultipleToGitExclude(tc.entries); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			entries := gitExcludeSectionEntries(t, GitExcludePath)
			if !slices.Equal(entries, tc.wantEntries) {
				t.Fatalf("unexpected section entries: got %v, want %v", entries, tc.wantEntries)
			}

			content, err := os.ReadFile(GitExcludePath)
			if err != nil {
				t.Fatalf("failed to read exclude file: %v", err)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(string(content), want) {
					t.Fatalf("exclude file does not contain %q:\n%s", want, content)
				}
			}
		})
	}
}
