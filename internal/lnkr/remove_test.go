package lnkr

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRemove(t *testing.T) {
	testCases := []struct {
		name          string
		remoteFiles   map[string]string
		localFiles    map[string]string // plain local files created without linking
		links         []Link
		makeLinks     bool // create local links pointing at the remote files
		removePath    string
		wantErr       bool
		wantRemaining []Link
		wantRestored  []string // local paths expected to be regular files afterwards
	}{
		{
			name:         "SymbolicFile",
			remoteFiles:  map[string]string{"a.txt": "content"},
			links:        []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			makeLinks:    true,
			removePath:   "a.txt",
			wantRestored: []string{"a.txt"},
		},
		{
			name:         "HardFile",
			remoteFiles:  map[string]string{"a.txt": "content"},
			links:        []Link{{Path: "a.txt", Type: LinkTypeHard}},
			makeLinks:    true,
			removePath:   "a.txt",
			wantRestored: []string{"a.txt"},
		},
		{
			name:         "NestedPathCleansEmptyRemoteDirs",
			remoteFiles:  map[string]string{"sub/dir/a.txt": "content"},
			links:        []Link{{Path: "sub/dir/a.txt", Type: LinkTypeSymbolic}},
			makeLinks:    true,
			removePath:   "sub/dir/a.txt",
			wantRestored: []string{"sub/dir/a.txt"},
		},
		{
			name: "DirectoryPrefixRemovesChildren",
			remoteFiles: map[string]string{
				"conf/a.txt": "a",
				"conf/b.txt": "b",
			},
			links: []Link{
				{Path: "conf/a.txt", Type: LinkTypeHard},
				{Path: "conf/b.txt", Type: LinkTypeHard},
			},
			makeLinks:    true,
			removePath:   "conf",
			wantRestored: []string{"conf/a.txt", "conf/b.txt"},
		},
		{
			name: "PrefixDoesNotMatchSibling",
			remoteFiles: map[string]string{
				"conf/a.txt": "a",
				"confab.txt": "c",
			},
			links: []Link{
				{Path: "conf/a.txt", Type: LinkTypeSymbolic},
				{Path: "confab.txt", Type: LinkTypeSymbolic},
			},
			makeLinks:     true,
			removePath:    "conf",
			wantRemaining: []Link{{Path: "confab.txt", Type: LinkTypeSymbolic}},
			wantRestored:  []string{"conf/a.txt"},
		},
		{
			name:          "NoMatch",
			remoteFiles:   map[string]string{"a.txt": "content"},
			links:         []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			makeLinks:     true,
			removePath:    "b.txt",
			wantRemaining: []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
		},
		{
			name:       "RemoteMissing",
			links:      []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			removePath: "a.txt",
			wantErr:    true,
		},
		{
			name:        "RegularFileWhereSymlinkExpected",
			remoteFiles: map[string]string{"a.txt": "remote"},
			localFiles:  map[string]string{"a.txt": "local"},
			links:       []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
			removePath:  "a.txt",
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localDir, remoteDir := setupProject(t, &Config{Links: tc.links})
			writeFiles(t, remoteDir, tc.remoteFiles)
			writeFiles(t, localDir, tc.localFiles)

			if tc.makeLinks {
				for _, link := range tc.links {
					localPath := filepath.Join(localDir, link.Path)
					remotePath := filepath.Join(remoteDir, link.Path)
					if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
						t.Fatalf("failed to create local parent dir: %v", err)
					}
					if err := createLink(remotePath, localPath, link.Type); err != nil {
						t.Fatalf("failed to create link: %v", err)
					}
				}
			}

			err := Remove(tc.removePath)
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
			if !slices.Equal(config.Links, tc.wantRemaining) {
				t.Fatalf("unexpected remaining links: got %+v, want %+v", config.Links, tc.wantRemaining)
			}

			// Restored files must be regular local files with the remote gone.
			for _, path := range tc.wantRestored {
				fi, err := os.Lstat(filepath.Join(localDir, path))
				if err != nil {
					t.Fatalf("restored file missing: %v", err)
				}
				if fi.Mode()&os.ModeSymlink != 0 {
					t.Fatalf("restored file is still a symlink: %s", path)
				}
				content, err := os.ReadFile(filepath.Join(localDir, path))
				if err != nil {
					t.Fatalf("failed to read restored file: %v", err)
				}
				if string(content) != tc.remoteFiles[path] {
					t.Fatalf("unexpected restored content: got %q, want %q", content, tc.remoteFiles[path])
				}
				if _, err := os.Stat(filepath.Join(remoteDir, path)); !os.IsNotExist(err) {
					t.Fatalf("remote file still exists: %s", path)
				}
			}
		})
	}
}

func TestRemoveCleansEmptyRemoteDirs(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{
		Links: []Link{{Path: "sub/dir/a.txt", Type: LinkTypeSymbolic}},
	})
	writeFiles(t, remoteDir, map[string]string{"sub/dir/a.txt": "content"})

	localPath := filepath.Join(localDir, "sub", "dir", "a.txt")
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		t.Fatalf("failed to create local parent dir: %v", err)
	}
	if err := createLink(filepath.Join(remoteDir, "sub", "dir", "a.txt"), localPath, LinkTypeSymbolic); err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	if err := Remove("sub/dir/a.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Emptied parent directories in remote must be cleaned up.
	if _, err := os.Stat(filepath.Join(remoteDir, "sub")); !os.IsNotExist(err) {
		t.Fatalf("empty remote directory was not cleaned up")
	}
	// The remote root itself must remain.
	if _, err := os.Stat(remoteDir); err != nil {
		t.Fatalf("remote root was removed: %v", err)
	}
}
