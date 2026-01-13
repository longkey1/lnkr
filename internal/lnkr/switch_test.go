package lnkr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSwitch(t *testing.T) {
	testCases := []struct {
		name        string
		initialType string
		newType     string
		wantType    string
		wantErr     bool
	}{
		{
			name:        "SymbolicToHard",
			initialType: LinkTypeSymbolic,
			newType:     LinkTypeHard,
			wantType:    LinkTypeHard,
		},
		{
			name:        "HardToSymbolic",
			initialType: LinkTypeHard,
			newType:     LinkTypeSymbolic,
			wantType:    LinkTypeSymbolic,
		},
		{
			name:        "ToggleFromSymbolic",
			initialType: LinkTypeSymbolic,
			newType:     "",
			wantType:    LinkTypeHard,
		},
		{
			name:        "ToggleFromHard",
			initialType: LinkTypeHard,
			newType:     "",
			wantType:    LinkTypeSymbolic,
		},
		{
			name:        "SameTypeSymbolic",
			initialType: LinkTypeSymbolic,
			newType:     LinkTypeSymbolic,
			wantType:    LinkTypeSymbolic,
		},
		{
			name:        "InvalidType",
			initialType: LinkTypeSymbolic,
			newType:     "invalid",
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			localDir := filepath.Join(tempDir, "local")
			remoteDir := filepath.Join(tempDir, "remote")

			if err := os.MkdirAll(localDir, 0755); err != nil {
				t.Fatalf("failed to create local dir: %v", err)
			}
			if err := os.MkdirAll(remoteDir, 0755); err != nil {
				t.Fatalf("failed to create remote dir: %v", err)
			}

			// Create test file in remote
			testFile := "test.txt"
			remotePath := filepath.Join(remoteDir, testFile)
			if err := os.WriteFile(remotePath, []byte("test content"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Create initial link
			localPath := filepath.Join(localDir, testFile)
			if err := createLink(remotePath, localPath, tc.initialType); err != nil {
				t.Fatalf("failed to create initial link: %v", err)
			}

			// Save working directory
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

			// Create config file
			config := &Config{
				Local:  localDir,
				Remote: remoteDir,
				Links: []Link{
					{Path: testFile, Type: tc.initialType},
				},
			}
			if err := saveConfig(config); err != nil {
				t.Fatalf("failed to save config: %v", err)
			}

			// Run switch
			err = Switch(testFile, tc.newType)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Reload config and check
			config, err = loadConfig()
			if err != nil {
				t.Fatalf("failed to reload config: %v", err)
			}

			if len(config.Links) != 1 {
				t.Fatalf("unexpected number of links: %d", len(config.Links))
			}

			if config.Links[0].Type != tc.wantType {
				t.Fatalf("unexpected link type: got %q, want %q", config.Links[0].Type, tc.wantType)
			}

			// Verify link exists
			if _, err := os.Lstat(localPath); err != nil {
				t.Fatalf("link does not exist after switch: %v", err)
			}
		})
	}
}

func TestSwitchDirectorySymToHard(t *testing.T) {
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	remoteDir := filepath.Join(tempDir, "remote")

	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}

	// Create test directory in remote with files
	testDir := "testdir"
	remoteTestDir := filepath.Join(remoteDir, testDir)
	if err := os.MkdirAll(remoteTestDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	// Add files to the directory
	if err := os.WriteFile(filepath.Join(remoteTestDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(remoteTestDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Create symbolic link to directory
	localTestDir := filepath.Join(localDir, testDir)
	if err := os.Symlink(remoteTestDir, localTestDir); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Save working directory
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

	// Create config file
	config := &Config{
		Local:  localDir,
		Remote: remoteDir,
		Links: []Link{
			{Path: testDir, Type: LinkTypeSymbolic},
		},
	}
	if err := saveConfig(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Switch directory sym -> hard (recursive)
	if err := Switch(testDir, LinkTypeHard); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config has file entries instead of directory
	config, err = loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	// Should have 2 file entries now
	if len(config.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(config.Links))
	}

	// Verify hard links exist
	for _, link := range config.Links {
		if link.Type != LinkTypeHard {
			t.Fatalf("expected hard link type, got %s", link.Type)
		}
		localFile := filepath.Join(localDir, link.Path)
		fi, err := os.Lstat(localFile)
		if err != nil {
			t.Fatalf("file not found: %s", localFile)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("expected hard link, got symlink: %s", localFile)
		}
	}
}

func TestSwitchDirectoryHardToSym(t *testing.T) {
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	remoteDir := filepath.Join(tempDir, "remote")

	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}

	// Create test directory in remote with files
	testDir := "testdir"
	remoteTestDir := filepath.Join(remoteDir, testDir)
	if err := os.MkdirAll(remoteTestDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(remoteTestDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(remoteTestDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Create local directory with hard links
	localTestDir := filepath.Join(localDir, testDir)
	if err := os.MkdirAll(localTestDir, 0755); err != nil {
		t.Fatalf("failed to create local test dir: %v", err)
	}
	if err := os.Link(filepath.Join(remoteTestDir, "file1.txt"), filepath.Join(localTestDir, "file1.txt")); err != nil {
		t.Fatalf("failed to create hard link: %v", err)
	}
	if err := os.Link(filepath.Join(remoteTestDir, "file2.txt"), filepath.Join(localTestDir, "file2.txt")); err != nil {
		t.Fatalf("failed to create hard link: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalWD)
	})

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create config with hard link file entries
	config := &Config{
		Local:  localDir,
		Remote: remoteDir,
		Links: []Link{
			{Path: filepath.Join(testDir, "file1.txt"), Type: LinkTypeHard},
			{Path: filepath.Join(testDir, "file2.txt"), Type: LinkTypeHard},
		},
	}
	if err := saveConfig(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Switch directory hard -> sym
	if err := Switch(testDir, LinkTypeSymbolic); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config has single directory entry
	config, err = loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if len(config.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(config.Links))
	}

	if config.Links[0].Path != testDir || config.Links[0].Type != LinkTypeSymbolic {
		t.Fatalf("expected sym link for %s, got %v", testDir, config.Links[0])
	}

	// Verify symlink exists
	fi, err := os.Lstat(localTestDir)
	if err != nil {
		t.Fatalf("local dir not found: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got regular file/dir")
	}
}

func TestSwitchNotFound(t *testing.T) {
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	remoteDir := filepath.Join(tempDir, "remote")

	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}

	// Save working directory
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

	// Create config file with no links
	config := &Config{
		Local:  localDir,
		Remote: remoteDir,
		Links:  []Link{},
	}
	if err := saveConfig(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Try to switch non-existent path - should fail
	err = Switch("nonexistent.txt", LinkTypeHard)
	if err == nil {
		t.Fatalf("expected error when switching non-existent path, but got none")
	}
}
