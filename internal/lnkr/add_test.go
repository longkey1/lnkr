package lnkr

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// setupProject creates a temporary project directory with local and remote
// subdirectories, changes the working directory to the project root, and
// saves the given configuration as .lnkr.toml when provided.
// It returns the local and remote directory paths.
func setupProject(t *testing.T, config *Config) (localDir, remoteDir string) {
	t.Helper()

	tempDir := t.TempDir()
	localDir = filepath.Join(tempDir, "local")
	remoteDir = filepath.Join(tempDir, "remote")

	for _, dir := range []string{localDir, remoteDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	t.Chdir(tempDir)

	if config != nil {
		config.Local = localDir
		config.Remote = remoteDir
		if err := saveConfig(config); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	}

	return localDir, remoteDir
}

// writeFiles creates files (relative path -> content) under baseDir,
// creating parent directories as needed.
func writeFiles(t *testing.T, baseDir string, files map[string]string) {
	t.Helper()

	for path, content := range files {
		full := filepath.Join(baseDir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("failed to create parent dir for %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", full, err)
		}
	}
}

// assertLink verifies that localPath is a link of the given type pointing to remotePath.
func assertLink(t *testing.T, localPath, remotePath, linkType string) {
	t.Helper()

	fi, err := os.Lstat(localPath)
	if err != nil {
		t.Fatalf("link does not exist at %s: %v", localPath, err)
	}
	remoteInfo, err := os.Stat(remotePath)
	if err != nil {
		t.Fatalf("remote path does not exist at %s: %v", remotePath, err)
	}

	switch linkType {
	case LinkTypeSymbolic:
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("expected symbolic link at %s", localPath)
		}
		target, err := os.Readlink(localPath)
		if err != nil {
			t.Fatalf("failed to read link %s: %v", localPath, err)
		}
		if target != remotePath {
			t.Fatalf("unexpected link target: got %q, want %q", target, remotePath)
		}
	case LinkTypeHard:
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("expected hard link, got symlink: %s", localPath)
		}
		if !os.SameFile(fi, remoteInfo) {
			t.Fatalf("expected %s and %s to be the same file", localPath, remotePath)
		}
	default:
		t.Fatalf("unknown link type: %s", linkType)
	}
}

// gitExcludeSectionEntries returns the entries between the LNKR markers in the exclude file.
func gitExcludeSectionEntries(t *testing.T, excludePath string) []string {
	t.Helper()

	content, err := os.ReadFile(excludePath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", excludePath, err)
	}

	var entries []string
	inSection := false
	for line := range strings.SplitSeq(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case GitExcludeSectionStart:
			inSection = true
		case GitExcludeSectionEnd:
			inSection = false
		default:
			if inSection && trimmed != "" {
				entries = append(entries, trimmed)
			}
		}
	}
	return entries
}

func TestAdd(t *testing.T) {
	testCases := []struct {
		name      string
		files     map[string]string
		addPath   string
		recursive bool
		linkType  string
		wantErr   bool
		wantLinks []Link
	}{
		{
			name:      "SymbolicFile",
			files:     map[string]string{"notes.txt": "content"},
			addPath:   "notes.txt",
			linkType:  LinkTypeSymbolic,
			wantLinks: []Link{{Path: "notes.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:      "HardFile",
			files:     map[string]string{"notes.txt": "content"},
			addPath:   "notes.txt",
			linkType:  LinkTypeHard,
			wantLinks: []Link{{Path: "notes.txt", Type: LinkTypeHard}},
		},
		{
			name:      "SymbolicAliasNormalized",
			files:     map[string]string{"notes.txt": "content"},
			addPath:   "notes.txt",
			linkType:  "symbolic",
			wantLinks: []Link{{Path: "notes.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:      "SymbolicDirectory",
			files:     map[string]string{"conf/a.txt": "a", "conf/b.txt": "b"},
			addPath:   "conf",
			linkType:  LinkTypeSymbolic,
			wantLinks: []Link{{Path: "conf", Type: LinkTypeSymbolic}},
		},
		{
			name:      "HardDirectoryRecursive",
			files:     map[string]string{"conf/b.txt": "b", "conf/a.txt": "a"},
			addPath:   "conf",
			recursive: true,
			linkType:  LinkTypeHard,
			wantLinks: []Link{
				{Path: "conf/a.txt", Type: LinkTypeHard},
				{Path: "conf/b.txt", Type: LinkTypeHard},
			},
		},
		{
			name:     "InvalidLinkType",
			files:    map[string]string{"notes.txt": "content"},
			addPath:  "notes.txt",
			linkType: "invalid",
			wantErr:  true,
		},
		{
			name:     "AbsolutePath",
			files:    map[string]string{"notes.txt": "content"},
			addPath:  "/notes.txt",
			linkType: LinkTypeSymbolic,
			wantErr:  true,
		},
		{
			name:     "HardDirectoryWithoutRecursive",
			files:    map[string]string{"conf/a.txt": "a"},
			addPath:  "conf",
			linkType: LinkTypeHard,
			wantErr:  true,
		},
		{
			name:      "RecursiveWithSymbolic",
			files:     map[string]string{"notes.txt": "content"},
			addPath:   "notes.txt",
			recursive: true,
			linkType:  LinkTypeSymbolic,
			wantErr:   true,
		},
		{
			name:     "PathDoesNotExist",
			addPath:  "missing.txt",
			linkType: LinkTypeSymbolic,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localDir, remoteDir := setupProject(t, &Config{Links: []Link{}})
			writeFiles(t, localDir, tc.files)

			err := Add(tc.addPath, tc.recursive, tc.linkType, false)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			config, err := loadConfig()
			if err != nil {
				t.Fatalf("failed to reload config: %v", err)
			}
			if !slices.Equal(config.Links, tc.wantLinks) {
				t.Fatalf("unexpected links: got %+v, want %+v", config.Links, tc.wantLinks)
			}

			// Every link must exist at local, pointing to the moved remote file.
			for _, link := range tc.wantLinks {
				assertLink(t, filepath.Join(localDir, link.Path), filepath.Join(remoteDir, link.Path), link.Type)
			}

			// Git exclude must contain .lnkr.toml and all link paths.
			entries := gitExcludeSectionEntries(t, GitExcludePath)
			if !slices.Contains(entries, "/"+ConfigFileName) {
				t.Fatalf("git exclude does not contain /%s: %v", ConfigFileName, entries)
			}
			for _, link := range tc.wantLinks {
				if !slices.Contains(entries, "/"+link.Path) {
					t.Fatalf("git exclude does not contain /%s: %v", link.Path, entries)
				}
			}
		})
	}
}

func TestAddDuplicate(t *testing.T) {
	localDir, _ := setupProject(t, &Config{Links: []Link{}})
	writeFiles(t, localDir, map[string]string{"notes.txt": "content"})

	if err := Add("notes.txt", false, LinkTypeSymbolic, false); err != nil {
		t.Fatalf("unexpected error on first add: %v", err)
	}

	// Second add is a no-op because the path is already registered.
	if err := Add("notes.txt", false, LinkTypeSymbolic, false); err != nil {
		t.Fatalf("unexpected error on duplicate add: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(config.Links) != 1 {
		t.Fatalf("expected 1 link after duplicate add, got %d", len(config.Links))
	}
}

func TestAddFromSubdirectory(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{Links: []Link{}})
	writeFiles(t, localDir, map[string]string{"conf/a.txt": "a"})

	// Paths are resolved relative to the current directory.
	t.Chdir(filepath.Join(localDir, "conf"))

	if err := Add("a.txt", false, LinkTypeSymbolic, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	want := []Link{{Path: "conf/a.txt", Type: LinkTypeSymbolic}}
	if !slices.Equal(config.Links, want) {
		t.Fatalf("unexpected links: got %+v, want %+v", config.Links, want)
	}
	assertLink(t, filepath.Join(localDir, "conf/a.txt"), filepath.Join(remoteDir, "conf/a.txt"), LinkTypeSymbolic)
}

func TestAddAbsolutePathInsideLocal(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{Links: []Link{}})
	writeFiles(t, localDir, map[string]string{"notes.txt": "content"})

	if err := Add(filepath.Join(localDir, "notes.txt"), false, LinkTypeSymbolic, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	want := []Link{{Path: "notes.txt", Type: LinkTypeSymbolic}}
	if !slices.Equal(config.Links, want) {
		t.Fatalf("unexpected links: got %+v, want %+v", config.Links, want)
	}
	assertLink(t, filepath.Join(localDir, "notes.txt"), filepath.Join(remoteDir, "notes.txt"), LinkTypeSymbolic)
}

func TestAddDryRun(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{Links: []Link{}})
	writeFiles(t, localDir, map[string]string{"notes.txt": "content"})

	if err := Add("notes.txt", false, LinkTypeSymbolic, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nothing must change: no move, no link, no config entry.
	fi, err := os.Lstat(filepath.Join(localDir, "notes.txt"))
	if err != nil {
		t.Fatalf("local file missing after dry run: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("local file was replaced by a link during dry run")
	}
	if _, err := os.Stat(filepath.Join(remoteDir, "notes.txt")); !os.IsNotExist(err) {
		t.Fatalf("file was moved to remote during dry run")
	}
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(config.Links) != 0 {
		t.Fatalf("config was modified during dry run: %+v", config.Links)
	}
}

func TestAddUnconfigured(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name:   "LocalNotSet",
			config: &Config{Remote: "/tmp/remote"},
		},
		{
			name:   "RemoteNotSet",
			config: &Config{Local: "/tmp/local"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			if err := saveConfig(tc.config); err != nil {
				t.Fatalf("failed to save config: %v", err)
			}

			if err := Add("notes.txt", false, LinkTypeSymbolic, false); err == nil {
				t.Fatalf("expected error but got none")
			}
		})
	}
}
