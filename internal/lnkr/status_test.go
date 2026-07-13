package lnkr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToPlaceholderPath(t *testing.T) {
	testCases := []struct {
		name        string
		fullPath    string
		rootPath    string
		placeholder string
		want        string
	}{
		{
			name:        "ReplacesRootPrefix",
			fullPath:    "/home/user/project/a.txt",
			rootPath:    "/home/user/project",
			placeholder: "{local}",
			want:        "{local}/a.txt",
		},
		{
			name:        "ExactRootMatch",
			fullPath:    "/home/user/project",
			rootPath:    "/home/user/project",
			placeholder: "{local}",
			want:        "{local}",
		},
		{
			name:        "NoMatchReturnsFullPath",
			fullPath:    "/other/place/a.txt",
			rootPath:    "/home/user/project",
			placeholder: "{local}",
			want:        "/other/place/a.txt",
		},
		{
			name:        "EmptyRootReturnsFullPath",
			fullPath:    "/home/user/project/a.txt",
			rootPath:    "",
			placeholder: "{local}",
			want:        "/home/user/project/a.txt",
		},
		{
			name:        "EmptyFullPathReturnsEmpty",
			fullPath:    "",
			rootPath:    "/home/user/project",
			placeholder: "{local}",
			want:        "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := toPlaceholderPath(tc.fullPath, tc.rootPath, tc.placeholder)
			if got != tc.want {
				t.Fatalf("toPlaceholderPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetStatusText(t *testing.T) {
	testCases := []struct {
		name   string
		status LinkStatus
		want   string
	}{
		{
			name:   "NotExists",
			status: LinkStatus{Exists: false},
			want:   "LINK NOT FOUND",
		},
		{
			name:   "ExistsWithError",
			status: LinkStatus{Exists: true, Error: "Not a symbolic link"},
			want:   "Not a symbolic link",
		},
		{
			name:   "Linked",
			status: LinkStatus{Exists: true, IsLink: true},
			want:   "LINKED",
		},
		{
			name:   "ExistsButNotLinked",
			status: LinkStatus{Exists: true, IsLink: false},
			want:   "NOT LINKED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getStatusText(tc.status)
			if got != tc.want {
				t.Fatalf("getStatusText() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCheckLinkStatus(t *testing.T) {
	testCases := []struct {
		name            string
		setup           func(t *testing.T, localDir, remoteDir string)
		link            Link
		noLocal         bool // leave config.Local empty
		noRemote        bool // leave config.Remote empty
		wantExists      bool
		wantIsLink      bool
		wantErrContains string // empty means no error expected
	}{
		{
			name: "SymbolicLinked",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})
				if err := os.Symlink(filepath.Join(remoteDir, "a.txt"), filepath.Join(localDir, "a.txt")); err != nil {
					t.Fatalf("failed to create symlink: %v", err)
				}
			},
			link:       Link{Path: "a.txt", Type: LinkTypeSymbolic},
			wantExists: true,
			wantIsLink: true,
		},
		{
			name: "SymbolicWrongTarget",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"a.txt": "a", "other.txt": "o"})
				if err := os.Symlink(filepath.Join(remoteDir, "other.txt"), filepath.Join(localDir, "a.txt")); err != nil {
					t.Fatalf("failed to create symlink: %v", err)
				}
			},
			link:            Link{Path: "a.txt", Type: LinkTypeSymbolic},
			wantExists:      true,
			wantErrContains: "Wrong target",
		},
		{
			name: "SymbolicNotASymlink",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})
				writeFiles(t, localDir, map[string]string{"a.txt": "a"})
			},
			link:            Link{Path: "a.txt", Type: LinkTypeSymbolic},
			wantExists:      true,
			wantErrContains: "Not a symbolic link",
		},
		{
			name: "SymbolicTargetMissing",
			setup: func(t *testing.T, localDir, remoteDir string) {
				if err := os.Symlink(filepath.Join(remoteDir, "a.txt"), filepath.Join(localDir, "a.txt")); err != nil {
					t.Fatalf("failed to create symlink: %v", err)
				}
			},
			link:            Link{Path: "a.txt", Type: LinkTypeSymbolic},
			wantExists:      true,
			wantErrContains: "TARGET NOT FOUND",
		},
		{
			name:            "LinkNotFound",
			setup:           func(t *testing.T, localDir, remoteDir string) {},
			link:            Link{Path: "a.txt", Type: LinkTypeSymbolic},
			wantExists:      false,
			wantErrContains: "LINK NOT FOUND",
		},
		{
			name: "HardLinked",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})
				if err := os.Link(filepath.Join(remoteDir, "a.txt"), filepath.Join(localDir, "a.txt")); err != nil {
					t.Fatalf("failed to create hard link: %v", err)
				}
			},
			link:       Link{Path: "a.txt", Type: LinkTypeHard},
			wantExists: true,
			wantIsLink: true,
		},
		{
			name: "HardDifferentInode",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"a.txt": "a"})
				writeFiles(t, localDir, map[string]string{"a.txt": "a"})
			},
			link:            Link{Path: "a.txt", Type: LinkTypeHard},
			wantExists:      true,
			wantErrContains: "Not a hard link",
		},
		{
			name: "HardTargetMissing",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, localDir, map[string]string{"a.txt": "a"})
			},
			link:            Link{Path: "a.txt", Type: LinkTypeHard},
			wantExists:      true,
			wantErrContains: "TARGET NOT FOUND",
		},
		{
			name: "HardDirectoryLinked",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"conf/a.txt": "a", "conf/b.txt": "b"})
				if err := os.MkdirAll(filepath.Join(localDir, "conf"), 0755); err != nil {
					t.Fatalf("failed to create local dir: %v", err)
				}
				for _, name := range []string{"a.txt", "b.txt"} {
					if err := os.Link(filepath.Join(remoteDir, "conf", name), filepath.Join(localDir, "conf", name)); err != nil {
						t.Fatalf("failed to create hard link: %v", err)
					}
				}
			},
			link:       Link{Path: "conf", Type: LinkTypeHard},
			wantExists: true,
			wantIsLink: true,
		},
		{
			name: "HardDirectoryNotLinked",
			setup: func(t *testing.T, localDir, remoteDir string) {
				writeFiles(t, remoteDir, map[string]string{"conf/a.txt": "a"})
				writeFiles(t, localDir, map[string]string{"conf/a.txt": "a"})
			},
			link:            Link{Path: "conf", Type: LinkTypeHard},
			wantExists:      true,
			wantErrContains: "not hard linked",
		},
		{
			name:            "LocalNotConfigured",
			setup:           func(t *testing.T, localDir, remoteDir string) {},
			link:            Link{Path: "a.txt", Type: LinkTypeSymbolic},
			noLocal:         true,
			wantErrContains: "Local directory not configured",
		},
		{
			name:            "RemoteNotConfigured",
			setup:           func(t *testing.T, localDir, remoteDir string) {},
			link:            Link{Path: "a.txt", Type: LinkTypeSymbolic},
			noRemote:        true,
			wantErrContains: "Remote directory not configured",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			localDir := filepath.Join(tempDir, "local")
			remoteDir := filepath.Join(tempDir, "remote")
			for _, dir := range []string{localDir, remoteDir} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create dir %s: %v", dir, err)
				}
			}

			tc.setup(t, localDir, remoteDir)

			config := &Config{Local: localDir, Remote: remoteDir}
			if tc.noLocal {
				config.Local = ""
			}
			if tc.noRemote {
				config.Remote = ""
			}

			status := checkLinkStatus(tc.link, config)

			if status.Exists != tc.wantExists {
				t.Fatalf("unexpected Exists: got %v, want %v", status.Exists, tc.wantExists)
			}
			if status.IsLink != tc.wantIsLink {
				t.Fatalf("unexpected IsLink: got %v, want %v", status.IsLink, tc.wantIsLink)
			}
			if tc.wantErrContains == "" {
				if status.Error != "" {
					t.Fatalf("unexpected error: %q", status.Error)
				}
			} else if !strings.Contains(status.Error, tc.wantErrContains) {
				t.Fatalf("unexpected error: got %q, want it to contain %q", status.Error, tc.wantErrContains)
			}
		})
	}
}
