package lnkr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLocalRelPath(t *testing.T) {
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	writeFiles(t, localDir, map[string]string{
		"notes.txt":  "n",
		"conf/a.txt": "a",
	})

	testCases := []struct {
		name    string
		chdir   string // relative to tempDir; empty means tempDir itself
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "RelativeFromLocalRoot",
			chdir: "local",
			input: "notes.txt",
			want:  "notes.txt",
		},
		{
			name:  "RelativeFromSubdirectory",
			chdir: "local/conf",
			input: "a.txt",
			want:  "conf/a.txt",
		},
		{
			name:  "DotSlashPrefix",
			chdir: "local",
			input: "./notes.txt",
			want:  "notes.txt",
		},
		{
			name:  "TrailingSlash",
			chdir: "local",
			input: "conf/",
			want:  "conf",
		},
		{
			name:  "AbsolutePathInsideLocal",
			chdir: "",
			input: filepath.Join(localDir, "conf", "a.txt"),
			want:  "conf/a.txt",
		},
		{
			name:  "OutsideCwdFallsBackToLocalRelative",
			chdir: "",
			input: "conf/a.txt",
			want:  "conf/a.txt",
		},
		{
			name:    "AbsolutePathOutsideLocal",
			chdir:   "",
			input:   filepath.Join(tempDir, "elsewhere.txt"),
			wantErr: true,
		},
		{
			name:    "LocalRootItselfRejected",
			chdir:   "local",
			input:   ".",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(filepath.Join(tempDir, tc.chdir))

			got, err := resolveLocalRelPath(tc.input, localDir)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveLocalRelPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestResolveLocalRelPathNonExistentInsideLocal(t *testing.T) {
	tempDir := t.TempDir()
	localDir := filepath.Join(tempDir, "local")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}
	t.Chdir(localDir)

	// A path that does not exist yet must still resolve (existence is checked
	// by the callers), so error messages can point at the right location.
	got, err := resolveLocalRelPath("missing.txt", localDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "missing.txt" {
		t.Fatalf("resolveLocalRelPath(missing.txt) = %q, want %q", got, "missing.txt")
	}
}
