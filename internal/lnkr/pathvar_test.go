package lnkr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	// Initialize viper for global config
	InitGlobalConfig()

	// Save and restore environment variables
	origHome := os.Getenv("HOME")
	origRemoteRoot := os.Getenv("LNKR_REMOTE_ROOT")
	defer func() {
		os.Setenv("HOME", origHome)
		if origRemoteRoot != "" {
			os.Setenv("LNKR_REMOTE_ROOT", origRemoteRoot)
		} else {
			os.Unsetenv("LNKR_REMOTE_ROOT")
		}
	}()

	// Set test environment variables
	os.Setenv("HOME", "/home/testuser")
	os.Setenv("LNKR_REMOTE_ROOT", "/remote/root")

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "absolute path without variables",
			path: "/absolute/path/to/file",
			want: "/absolute/path/to/file",
		},
		{
			name: "expand $HOME",
			path: "$HOME/.config/lnkr",
			want: "/home/testuser/.config/lnkr",
		},
		{
			name: "expand $LNKR_REMOTE_ROOT",
			path: "$LNKR_REMOTE_ROOT/project",
			want: "/remote/root/project",
		},
		{
			name: "expand ${HOME} with braces",
			path: "${HOME}/.config",
			want: "/home/testuser/.config",
		},
		{
			name: "combined variables",
			path: "$HOME/work/$LNKR_REMOTE_ROOT",
			want: "/home/testuser/work//remote/root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Clean paths for comparison
			wantClean := filepath.Clean(tt.want)
			if got != wantClean && tt.want != "" {
				t.Errorf("ExpandPath() = %v, want %v", got, wantClean)
			}
		})
	}
}

func TestExpandPath_UndefinedVariable(t *testing.T) {
	// Unset variable for test
	origValue := os.Getenv("UNDEFINED_VAR")
	os.Unsetenv("UNDEFINED_VAR")
	defer func() {
		if origValue != "" {
			os.Setenv("UNDEFINED_VAR", origValue)
		}
	}()

	_, err := ExpandPath("$UNDEFINED_VAR/path")
	if err == nil {
		t.Error("ExpandPath() expected error for undefined variable, got nil")
	}
}

func TestExpandPath_PWD(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	got, err := ExpandPath("$PWD/subdir")
	if err != nil {
		t.Errorf("ExpandPath() error = %v", err)
		return
	}

	want := filepath.Clean(filepath.Join(pwd, "subdir"))
	if got != want {
		t.Errorf("ExpandPath() = %v, want %v", got, want)
	}
}

func TestContractPath(t *testing.T) {
	// Initialize viper for global config
	InitGlobalConfig()

	// Save and restore environment variables
	origHome := os.Getenv("HOME")
	origRemoteRoot := os.Getenv("LNKR_REMOTE_ROOT")
	defer func() {
		os.Setenv("HOME", origHome)
		if origRemoteRoot != "" {
			os.Setenv("LNKR_REMOTE_ROOT", origRemoteRoot)
		} else {
			os.Unsetenv("LNKR_REMOTE_ROOT")
		}
	}()

	// Set test environment variables
	os.Setenv("HOME", "/home/testuser")
	os.Setenv("LNKR_REMOTE_ROOT", "/home/testuser/.config/lnkr")

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "path already has variable",
			path: "$HOME/.config",
			want: "$HOME/.config",
		},
		{
			name: "path already has placeholder",
			path: "{{remote_root}}/project",
			want: "{{remote_root}}/project",
		},
		{
			name: "contract to {{remote_root}} (more specific)",
			path: "/home/testuser/.config/lnkr/project",
			want: "{{remote_root}}/project",
		},
		{
			name: "contract to $HOME",
			path: "/home/testuser/documents",
			want: "$HOME/documents",
		},
		{
			name: "exact match $HOME",
			path: "/home/testuser",
			want: "$HOME",
		},
		{
			name: "exact match {{remote_root}}",
			path: "/home/testuser/.config/lnkr",
			want: "{{remote_root}}",
		},
		{
			name: "unrelated path stays unchanged",
			path: "/var/log/app",
			want: "/var/log/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContractPath(tt.path)
			if got != tt.want {
				t.Errorf("ContractPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContractPath_PWD(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Contract current directory should become $PWD
	got := ContractPath(pwd)
	if got != "$PWD" {
		t.Errorf("ContractPath(%s) = %v, want $PWD", pwd, got)
	}

	// Contract subdirectory of PWD
	subPath := filepath.Join(pwd, "subdir")
	got = ContractPath(subPath)
	if got != "$PWD/subdir" {
		t.Errorf("ContractPath(%s) = %v, want $PWD/subdir", subPath, got)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that absolute paths (old format) still work
	absPath := "/absolute/path/to/project"
	expanded, err := ExpandPath(absPath)
	if err != nil {
		t.Errorf("ExpandPath() should not error for absolute paths: %v", err)
	}
	if expanded != absPath {
		t.Errorf("ExpandPath() = %v, want %v (absolute paths should pass through)", expanded, absPath)
	}
}
