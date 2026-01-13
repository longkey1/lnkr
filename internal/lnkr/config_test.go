package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

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
			wantLinkType: "sym",
		},
		{
			name:         "Sym",
			linkType:     "sym",
			wantLinkType: "sym",
		},
		{
			name:         "Empty",
			linkType:     "",
			wantLinkType: "sym",
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
