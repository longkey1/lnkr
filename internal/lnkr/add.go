package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func Add(path string, recursive bool, linkType string, fromRemote bool) error {
	if linkType != LinkTypeHard && linkType != LinkTypeSymbolic {
		return fmt.Errorf("invalid link type: %s. Must be '%s' or '%s'", linkType, LinkTypeHard, LinkTypeSymbolic)
	}

	// Check if path is absolute
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is not allowed: %s. Please use relative path", path)
	}

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine base directory for relative paths
	var baseDir string
	if fromRemote {
		if config.Remote == "" {
			return fmt.Errorf("remote directory not configured. Run 'lnkr init --remote <path>' first")
		}
		baseDir = config.Remote
	} else {
		if config.Local == "" {
			return fmt.Errorf("local directory not configured. Run 'lnkr init --local <path>' first")
		}
		baseDir = config.Local
	}

	// Build absolute path and check if file exists
	absPath := filepath.Join(baseDir, path)
	fi, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	if recursive && linkType == LinkTypeSymbolic {
		return fmt.Errorf("recursive option cannot be used with symbolic links")
	}

	// Check existing links to avoid duplicates
	existing := make(map[string]struct{})
	for _, link := range config.Links {
		existing[link.Path] = struct{}{}
	}

	var targets []string

	// Add paths based on type and recursive flag
	if fi.IsDir() {
		if linkType == LinkTypeHard && !recursive {
			return fmt.Errorf("recursive option must be set when adding a directory with hard links")
		}

		if linkType == LinkTypeHard {
			// Walk directory and add all files for hard links
			err := filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return addPathToTargets(p, baseDir, existing, &targets)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}
		} else {
			// Add directory itself for symbolic links
			if err := addPathToTargets(absPath, baseDir, existing, &targets); err != nil {
				return err
			}
		}
	} else {
		// Add single file
		if err := addPathToTargets(absPath, baseDir, existing, &targets); err != nil {
			return err
		}
	}

	if len(targets) == 0 {
		fmt.Println("No new paths to add.")
		return nil
	}

	// Add links to config
	for _, t := range targets {
		config.Links = append(config.Links, Link{Path: t, Type: linkType})
		fmt.Printf("Added link: %s (type: %s)\n", t, linkType)
	}

	sort.Slice(config.Links, func(i, j int) bool {
		return config.Links[i].Path < config.Links[j].Path
	})

	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Apply all configured link paths to GitExclude
	if err := applyAllLinksToGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to apply link paths to GitExclude: %v\n", err)
	}

	return nil
}

// addPathToTargets adds a single path to the targets slice if it doesn't already exist
func addPathToTargets(absPath, baseDir string, existing map[string]struct{}, targets *[]string) error {
	relPath, err := filepath.Rel(baseDir, absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}
	if _, ok := existing[relPath]; !ok {
		*targets = append(*targets, relPath)
	}
	return nil
}
