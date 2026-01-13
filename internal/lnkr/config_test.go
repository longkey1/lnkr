package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigSourceValidation(t *testing.T) {
	// Cannot use t.Parallel() with os.Chdir()
	testCases := []struct {
		name       string
		source     string
		wantErr    bool
		wantSource string
	}{
		{
			name:       "Local",
			source:     "local",
			wantSource: "local",
		},
		{
			name:       "RemoteUppercase",
			source:     "REMOTE",
			wantSource: "remote",
		},
		{
			name:       "Empty",
			source:     "",
			wantSource: "local",
		},
		{
			name:    "Invalid",
			source:  "staging",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

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

			configContent := fmt.Sprintf(`local = "/tmp/local"
remote = "/tmp/remote"
source = "%s"
`, tc.source)

			configPath := filepath.Join(tempDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := loadConfig()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := cfg.GetSource()
			if got != tc.wantSource {
				t.Fatalf("unexpected source: got %q, want %q", got, tc.wantSource)
			}
		})
	}
}

func TestLoadConfigLinkTypeValidation(t *testing.T) {
	// Cannot use t.Parallel() with os.Chdir()
	testCases := []struct {
		name         string
		linkType     string
		wantErr      bool
		wantLinkType string
	}{
		{
			name:         "Hard",
			linkType:     "hard",
			wantLinkType: "hard",
		},
		{
			name:         "SymbolicUppercase",
			linkType:     "SYMBOLIC",
			wantLinkType: "symbolic",
		},
		{
			name:         "Empty",
			linkType:     "",
			wantLinkType: "hard",
		},
		{
			name:     "Invalid",
			linkType: "softlink",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

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

			configContent := fmt.Sprintf(`local = "/tmp/local"
remote = "/tmp/remote"
link_type = "%s"
`, tc.linkType)

			configPath := filepath.Join(tempDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := loadConfig()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := cfg.GetLinkType()
			if got != tc.wantLinkType {
				t.Fatalf("unexpected link_type: got %q, want %q", got, tc.wantLinkType)
			}
		})
	}
}
