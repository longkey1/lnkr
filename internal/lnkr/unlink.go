package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Unlink removes the links at local while keeping the entries in the
// configuration and the files in remote. Use 'lnkr link' to re-create them.
func Unlink(dryRun, assumeYes bool) error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(config.Links) == 0 {
		fmt.Printf("No links found in %s\n", ConfigFileName)
		return nil
	}

	// Use local directory as base for resolving link paths
	localDir, err := config.GetLocalExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand local path: %w", err)
	}
	remoteDir, err := config.GetRemoteExpanded()
	if err != nil {
		return fmt.Errorf("failed to expand remote path: %w", err)
	}

	if dryRun {
		for _, link := range config.Links {
			fmt.Printf("Would remove link: %s (type: %s)\n", filepath.Join(localDir, link.Path), link.Type)
		}
		fmt.Printf("Dry run: %d link(s) would be removed.\n", len(config.Links))
		return nil
	}

	if !assumeYes && !confirm(fmt.Sprintf("Remove %d link(s) under %s?", len(config.Links), localDir)) {
		fmt.Println("Aborted.")
		return nil
	}

	for _, link := range config.Links {
		if err := removeLinkEntry(link, localDir, remoteDir); err != nil {
			fmt.Printf("Error removing link for %s: %v\n", link.Path, err)
			continue
		}
	}

	// Remove all link paths from GitExclude
	if err := removeAllLinksFromGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to remove link paths from GitExclude: %v\n", err)
	}

	fmt.Println("Link removal completed.")
	return nil
}

func removeLinkEntry(link Link, localDir, remoteDir string) error {
	// Resolve absolute path for link
	linkAbs := filepath.Join(localDir, link.Path)

	if _, err := os.Lstat(linkAbs); os.IsNotExist(err) {
		fmt.Printf("Path does not exist, skipping: %s\n", linkAbs)
		return nil
	}

	switch link.Type {
	case LinkTypeHard:
		info, err := os.Stat(linkAbs)
		if err != nil {
			return fmt.Errorf("failed to stat path: %w", err)
		}

		if info.IsDir() {
			return removeHardLinkedDir(linkAbs, filepath.Join(remoteDir, link.Path))
		}
		if err := os.Remove(linkAbs); err != nil {
			return fmt.Errorf("failed to remove hard link: %w", err)
		}
		fmt.Printf("Removed hard link: %s\n", linkAbs)
	case LinkTypeSymbolic:
		if err := os.Remove(linkAbs); err != nil {
			return fmt.Errorf("failed to remove symbolic link: %w", err)
		}
		fmt.Printf("Removed symbolic link: %s\n", linkAbs)
	default:
		return fmt.Errorf("unknown link type: %s", link.Type)
	}

	return nil
}

// removeHardLinkedDir removes only the files that are hard links to the
// corresponding remote files. Unrelated files added after linking are kept so
// unlink never destroys data that exists nowhere else.
func removeHardLinkedDir(localDirPath, remoteDirPath string) error {
	var kept int
	err := filepath.Walk(localDirPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(localDirPath, p)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		remoteInfo, err := os.Stat(filepath.Join(remoteDirPath, relPath))
		if err == nil && os.SameFile(info, remoteInfo) {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("failed to remove hard link %s: %w", p, err)
			}
			fmt.Printf("Removed hard link: %s\n", p)
			return nil
		}

		kept++
		fmt.Printf("Kept (not linked to remote): %s\n", p)
		return nil
	})
	if err != nil {
		return err
	}

	removeEmptyDirTree(localDirPath)
	if kept > 0 {
		fmt.Printf("Kept %d file(s) under %s\n", kept, localDirPath)
	} else {
		fmt.Printf("Removed directory: %s\n", localDirPath)
	}
	return nil
}

// removeEmptyDirTree removes root and its subdirectories bottom-up,
// leaving any directory that still contains files.
func removeEmptyDirTree(root string) {
	var dirs []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		_ = os.Remove(d) // fails silently when not empty
	}
}

// removeAllLinksFromGitExclude removes the LNKR section from GitExclude
func removeAllLinksFromGitExclude(config *Config) error {
	excludePath := config.GetGitExcludePath()
	removed, err := removeGitExcludeSection(excludePath)
	if err != nil {
		return err
	}
	if removed {
		fmt.Printf("Removed all link paths from %s\n", excludePath)
	}
	return nil
}

// removeGitExcludeSection removes the LNKR section (including legacy markers)
// from the exclude file. It reports whether a section was removed.
func removeGitExcludeSection(excludePath string) (bool, error) {
	// Check if exclude file exists
	if _, err := os.Stat(excludePath); os.IsNotExist(err) {
		return false, nil
	}

	content, err := os.ReadFile(excludePath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")
	sectionStart, sectionEnd := findGitExcludeSection(lines)
	if sectionStart == -1 || sectionEnd == -1 {
		return false, nil
	}

	newLines := append(lines[:sectionStart], lines[sectionEnd+1:]...)
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(excludePath, []byte(newContent), 0644); err != nil {
		return false, err
	}

	return true, nil
}
