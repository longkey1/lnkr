package lnkr

import (
	"os"
	"path/filepath"
	"slices"
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

func TestCreateLinksSkipsExistingTarget(t *testing.T) {
	localDir, remoteDir := setupProject(t, &Config{
		Links: []Link{{Path: "a.txt", Type: LinkTypeSymbolic}},
	})
	writeFiles(t, remoteDir, map[string]string{"a.txt": "remote"})
	writeFiles(t, localDir, map[string]string{"a.txt": "local"})

	if err := CreateLinks(); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
