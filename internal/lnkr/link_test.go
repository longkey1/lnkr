package lnkr

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCreateLinks(t *testing.T) {
	testCases := []struct {
		name        string
		remoteFiles map[string]string
		links       []Link
		wantErr     bool
		wantLinked  []Link // links expected to exist at local after the call
	}{
		{
			name:        "SymbolicFile",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			wantLinked:  []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:        "HardFile",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeHard}},
			wantLinked:  []Link{{Path: "a.txt", Type: LinkTypeHard}},
		},
		{
			name:        "SymbolicDirectory",
			remoteFiles: map[string]string{"conf/a.txt": "a", "conf/b.txt": "b"},
			links:       []Link{{Path: "conf", Type: LinkTypeSymbolic}},
			wantLinked:  []Link{{Path: "conf", Type: LinkTypeSymbolic}},
		},
		{
			name:        "HardDirectoryRecursive",
			remoteFiles: map[string]string{"conf/a.txt": "a", "conf/sub/b.txt": "b"},
			links:       []Link{{Path: "conf", Type: LinkTypeHard}},
			wantLinked: []Link{
				{Path: "conf/a.txt", Type: LinkTypeHard},
				{Path: "conf/sub/b.txt", Type: LinkTypeHard},
			},
		},
		{
			name:        "NestedPathCreatesParentDirs",
			remoteFiles: map[string]string{"sub/dir/a.txt": "a"},
			links:       []Link{{Path: "sub/dir/a.txt", Type: LinkTypeSymbolic}},
			wantLinked:  []Link{{Path: "sub/dir/a.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:    "AllSourcesMissingFails",
			links:   []Link{{Path: "missing.txt", Type: LinkTypeSymbolic}},
			wantErr: true,
		},
		{
			name:        "UnknownLinkTypeFails",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: "bogus"}},
			wantErr:     true,
		},
		{
			name:        "PartialFailureSucceeds",
			remoteFiles: map[string]string{"a.txt": "a"},
			links: []Link{
				{Path: "a.txt", Type: LinkTypeSymbolic},
				{Path: "missing.txt", Type: LinkTypeSymbolic},
			},
			wantLinked: []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localDir, remoteDir := setupProject(t, &Config{Links: tc.links})
			writeFiles(t, remoteDir, tc.remoteFiles)

			err := CreateLinks()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, link := range tc.wantLinked {
				assertLink(t, filepath.Join(localDir, link.Path), filepath.Join(remoteDir, link.Path), link.Type)
			}

			// Git exclude must contain .lnkr.toml and all configured link paths.
			entries := gitExcludeSectionEntries(t, GitExcludePath)
			if !slices.Contains(entries, "/"+ConfigFileName) {
				t.Fatalf("git exclude does not contain /%s: %v", ConfigFileName, entries)
			}
			for _, link := range tc.links {
				if !slices.Contains(entries, "/"+link.Path) {
					t.Fatalf("git exclude does not contain /%s: %v", link.Path, entries)
				}
			}
		})
	}
}

func TestCreateLinksNoLinks(t *testing.T) {
	setupProject(t, &Config{Links: []Link{}})

	if err := CreateLinks(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateLinksFailsOnConflictingTarget(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{
		Links: []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
	})
	writeFiles(t, remoteDir, map[string]string{"a.txt": "remote"})
	writeFiles(t, localDir, map[string]string{"a.txt": "local"})

	// A local file that is not a link to remote must be reported as an error.
	if err := CreateLinks(); err == nil {
		t.Fatalf("expected error for conflicting target, but got none")
	}

	// The existing local file must be left untouched (not replaced by a link).
	fi, err := os.Lstat(filepath.Join(localDir, "a.txt"))
	if err != nil {
		t.Fatalf("failed to stat local file: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("existing target was replaced by a symlink")
	}
	content, err := os.ReadFile(filepath.Join(localDir, "a.txt"))
	if err != nil {
		t.Fatalf("failed to read local file: %v", err)
	}
	if string(content) != "local" {
		t.Fatalf("existing target content changed: got %q, want %q", content, "local")
	}
}

// TestCreateLinksFromConfigInArbitraryDir verifies the 1-remote : N-local
// model: placing the remote's canonical .lnkr.toml (as symlink or copy) into
// any directory makes 'lnkr link' create the links there.
func TestCreateLinksFromConfigInArbitraryDir(t *testing.T) {
	tempDir := t.TempDir()
	remoteDir := filepath.Join(tempDir, "remote")
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}
	writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})

	remoteConfig := newTestConfig(remoteDir, remoteDir, []Link{{Path: "a.txt", Type: LinkTypeSymbolic}})
	if err := saveConfig(remoteConfig); err != nil {
		t.Fatalf("failed to save remote config: %v", err)
	}
	remoteConfigPath := filepath.Join(remoteDir, ConfigFileName)

	for _, mode := range []string{"Symlink", "Copy"} {
		t.Run(mode, func(t *testing.T) {
			workDir := filepath.Join(tempDir, "work-"+mode)
			if err := os.MkdirAll(workDir, 0755); err != nil {
				t.Fatalf("failed to create work dir: %v", err)
			}

			workConfigPath := filepath.Join(workDir, ConfigFileName)
			switch mode {
			case "Symlink":
				if err := os.Symlink(remoteConfigPath, workConfigPath); err != nil {
					t.Fatalf("failed to symlink config: %v", err)
				}
			case "Copy":
				content, err := os.ReadFile(remoteConfigPath)
				if err != nil {
					t.Fatalf("failed to read remote config: %v", err)
				}
				if err := os.WriteFile(workConfigPath, content, 0644); err != nil {
					t.Fatalf("failed to copy config: %v", err)
				}
			}

			t.Chdir(workDir)
			if err := CreateLinks(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertLink(t, filepath.Join(workDir, "a.txt"), filepath.Join(remoteDir, "a.txt"), LinkTypeSymbolic)
		})
	}
}

// TestCreateLinksRefusesInsideRemote verifies the guard against running
// inside the remote directory, where links would be self-referential.
func TestCreateLinksRefusesInsideRemote(t *testing.T) {
	remoteDir := t.TempDir()
	writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})

	config := newTestConfig(remoteDir, remoteDir, []Link{{Path: "a.txt", Type: LinkTypeSymbolic}})
	if err := saveConfig(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	t.Chdir(remoteDir)
	err := CreateLinks()
	if err == nil {
		t.Fatalf("expected error when running inside remote, but got none")
	}
	if !strings.Contains(err.Error(), "remote directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLinkAndUnlinkSkipGitExcludeInWorktree verifies that a directory whose
// .git is a regular file (a git worktree) links and unlinks successfully,
// leaving .git untouched instead of failing on 'mkdir .git: not a directory'.
func TestLinkAndUnlinkSkipGitExcludeInWorktree(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{Links: []Link{{Path: "a.txt", Type: LinkTypeSymbolic}}})
	writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})

	gitFile := filepath.Join(localDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /elsewhere/.git/worktrees/x\n"), 0644); err != nil {
		t.Fatalf("failed to write .git file: %v", err)
	}

	if err := CreateLinks(); err != nil {
		t.Fatalf("unexpected error on link: %v", err)
	}
	assertLink(t, filepath.Join(localDir, "a.txt"), filepath.Join(remoteDir, "a.txt"), LinkTypeSymbolic)

	if err := Unlink(false, true); err != nil {
		t.Fatalf("unexpected error on unlink: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(localDir, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("link still exists after unlink")
	}

	// .git must remain the worktree pointer file.
	fi, err := os.Lstat(gitFile)
	if err != nil {
		t.Fatalf("failed to stat .git: %v", err)
	}
	if !fi.Mode().IsRegular() {
		t.Fatalf("expected .git to remain a regular file, got %v", fi.Mode())
	}
}

func TestCreateLinksIdempotent(t *testing.T) {
	testCases := []struct {
		name        string
		remoteFiles map[string]string
		links       []Link
		wantLinked  []Link
	}{
		{
			name:        "SymbolicFile",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			wantLinked:  []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:        "HardFile",
			remoteFiles: map[string]string{"a.txt": "a"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeHard}},
			wantLinked:  []Link{{Path: "a.txt", Type: LinkTypeHard}},
		},
		{
			name:        "HardDirectory",
			remoteFiles: map[string]string{"conf/a.txt": "a", "conf/b.txt": "b"},
			links:       []Link{{Path: "conf", Type: LinkTypeHard}},
			wantLinked: []Link{
				{Path: "conf/a.txt", Type: LinkTypeHard},
				{Path: "conf/b.txt", Type: LinkTypeHard},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localDir, remoteDir := setupProject(t, &Config{Links: tc.links})
			writeFiles(t, remoteDir, tc.remoteFiles)

			// Running twice must succeed with all links intact.
			if err := CreateLinks(); err != nil {
				t.Fatalf("unexpected error on first run: %v", err)
			}
			if err := CreateLinks(); err != nil {
				t.Fatalf("unexpected error on second run: %v", err)
			}

			for _, link := range tc.wantLinked {
				assertLink(t, filepath.Join(localDir, link.Path), filepath.Join(remoteDir, link.Path), link.Type)
			}
		})
	}
}
