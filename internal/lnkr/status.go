package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type LinkStatus struct {
	Path       string
	LocalPath  string
	RemotePath string
	Type       string
	Exists     bool
	IsLink     bool
	Error      string
}

func Status() error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Display root paths
	localRoot := config.Local
	remoteRoot := config.Remote
	if localRoot == "" {
		localRoot = "(not set)"
	}
	if remoteRoot == "" {
		remoteRoot = "(not set)"
	}
	fmt.Printf("Local Root:  %s\n", localRoot)
	fmt.Printf("Remote Root: %s\n", remoteRoot)
	fmt.Println()

	if len(config.Links) == 0 {
		fmt.Printf("No links found in %s\n", ConfigFileName)
		return nil
	}

	// Get expanded paths for display
	localExpanded, _ := config.GetLocalExpanded()
	remoteExpanded, _ := config.GetRemoteExpanded()

	var statuses []LinkStatus
	for _, link := range config.Links {
		status := checkLinkStatus(link, config)
		statuses = append(statuses, status)
	}

	// Calculate max width for each column using placeholder paths
	maxLocalPath := len("Local Path")
	maxRemotePath := len("Remote Path")
	maxType := len("Type")
	maxStatus := len("Status")
	for _, s := range statuses {
		localDisplay := toPlaceholderPath(s.LocalPath, localExpanded, "{local}")
		remoteDisplay := toPlaceholderPath(s.RemotePath, remoteExpanded, "{remote}")
		if len(localDisplay) > maxLocalPath {
			maxLocalPath = len(localDisplay)
		}
		if len(remoteDisplay) > maxRemotePath {
			maxRemotePath = len(remoteDisplay)
		}
		if len(s.Type) > maxType {
			maxType = len(s.Type)
		}
		st := getStatusText(s)
		if len(st) > maxStatus {
			maxStatus = len(st)
		}
	}

	// Print header
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s", maxLocalPath, "Local Path", maxRemotePath, "Remote Path", maxType, "Type", maxStatus, "Status")
	sep := strings.Repeat("-", len(header))
	fmt.Println(header)
	fmt.Println(sep)

	// Print each status with placeholder paths
	for _, s := range statuses {
		localDisplay := toPlaceholderPath(s.LocalPath, localExpanded, "{local}")
		remoteDisplay := toPlaceholderPath(s.RemotePath, remoteExpanded, "{remote}")
		st := getStatusText(s)
		fmt.Printf("%-*s  %-*s  %-*s  %-*s\n", maxLocalPath, localDisplay, maxRemotePath, remoteDisplay, maxType, s.Type, maxStatus, st)
	}

	return nil
}

// toPlaceholderPath replaces the root path with a placeholder
func toPlaceholderPath(fullPath, rootPath, placeholder string) string {
	if rootPath == "" || fullPath == "" {
		return fullPath
	}
	if strings.HasPrefix(fullPath, rootPath) {
		return placeholder + strings.TrimPrefix(fullPath, rootPath)
	}
	return fullPath
}

func getStatusText(status LinkStatus) string {
	if !status.Exists {
		return "LINK NOT FOUND"
	}
	if status.Error != "" {
		return status.Error
	}
	if status.IsLink {
		return "LINKED"
	}
	return "NOT LINKED"
}

func checkLinkStatus(link Link, config *Config) LinkStatus {
	status := LinkStatus{
		Path: link.Path,
		Type: link.Type,
	}

	// Validate config first
	if config.Local == "" {
		status.Error = "Local directory not configured"
		return status
	}
	if config.Remote == "" {
		status.Error = "Remote directory not configured"
		return status
	}

	// Expand paths with environment variables
	localDir, err := config.GetLocalExpanded()
	if err != nil {
		status.Error = fmt.Sprintf("Failed to expand local path: %v", err)
		return status
	}
	remoteDir, err := config.GetRemoteExpanded()
	if err != nil {
		status.Error = fmt.Sprintf("Failed to expand remote path: %v", err)
		return status
	}

	// Set the full paths
	status.LocalPath = filepath.Join(localDir, link.Path)
	status.RemotePath = filepath.Join(remoteDir, link.Path)

	// Check if the link path exists (use Lstat to not follow symlinks)
	info, err := os.Lstat(status.LocalPath)
	if os.IsNotExist(err) {
		status.Exists = false
		status.Error = "LINK NOT FOUND"
		return status
	}

	status.Exists = true

	switch link.Type {
	case LinkTypeSymbolic:
		// Check if it's actually a symbolic link
		if info.Mode()&os.ModeSymlink == 0 {
			status.Error = "Not a symbolic link"
			return status
		}

		// Get the target of the symbolic link
		target, err := os.Readlink(status.LocalPath)
		if err != nil {
			status.Error = fmt.Sprintf("Cannot read link target: %v", err)
			return status
		}

		// Check if the target exists
		if _, err := os.Stat(target); os.IsNotExist(err) {
			status.Error = "TARGET NOT FOUND"
			return status
		}

		// Check if the target path is correct (should point to remote location)
		if target != status.RemotePath {
			status.Error = fmt.Sprintf("Wrong target: %s (expected: %s)", target, status.RemotePath)
			return status
		}

		status.IsLink = true

	case LinkTypeHard:
		// Check if it's a directory
		if info.IsDir() {
			// For directories, check recursively that all files are hard linked
			err := checkHardLinksRecursively(status.LocalPath, status.RemotePath)
			if err != nil {
				status.Error = err.Error()
				return status
			}
			status.IsLink = true
			return status
		}

		// Check if the target file exists
		targetInfo, err := os.Stat(status.RemotePath)
		if os.IsNotExist(err) {
			status.Error = "TARGET NOT FOUND"
			return status
		}
		if err != nil {
			status.Error = fmt.Sprintf("Cannot access target file: %v", err)
			return status
		}

		// Check if both files have the same inode (hard link check)
		if info.Sys() == nil || targetInfo.Sys() == nil {
			status.Error = "Cannot get file system info for inode comparison"
			return status
		}

		// Compare inodes to verify hard link
		linkInode := getInode(info)
		targetInode := getInode(targetInfo)

		if linkInode == 0 || targetInode == 0 {
			status.Error = "Cannot determine inode numbers"
			return status
		}

		if linkInode != targetInode {
			status.Error = "Not a hard link (different inodes)"
			return status
		}

		status.IsLink = true
	}

	return status
}

// checkHardLinksRecursively verifies that all files in localDir are hard linked to corresponding files in remoteDir
func checkHardLinksRecursively(localDir, remoteDir string) error {
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localDir, localPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		remotePath := filepath.Join(remoteDir, relPath)

		// Check if remote file exists
		remoteInfo, err := os.Stat(remotePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("remote file not found: %s", remotePath)
		}
		if err != nil {
			return fmt.Errorf("cannot access remote file %s: %w", remotePath, err)
		}

		// Compare inodes
		localInode := getInode(info)
		remoteInode := getInode(remoteInfo)

		if localInode == 0 || remoteInode == 0 {
			return fmt.Errorf("cannot determine inode for %s", localPath)
		}

		if localInode != remoteInode {
			return fmt.Errorf("file %s is not hard linked (different inodes)", relPath)
		}

		return nil
	})
}

func getInode(fileInfo os.FileInfo) uint64 {
	sys := fileInfo.Sys()
	if sys == nil {
		return 0
	}

	switch sys := sys.(type) {
	case *syscall.Stat_t:
		return sys.Ino
	default:
		return 0
	}
}
