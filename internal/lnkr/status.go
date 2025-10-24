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

	// Print source/direction information from config
	src := config.GetSource()
	direction := "local -> remote"
	if strings.EqualFold(src, "remote") {
		direction = "remote -> local"
	}
	fmt.Printf("Source: %s (%s)\n\n", src, direction)

	if len(config.Links) == 0 {
		fmt.Printf("No links found in %s\n", ConfigFileName)
		return nil
	}

	var statuses []LinkStatus
	for _, link := range config.Links {
		status := checkLinkStatus(link, config)
		statuses = append(statuses, status)
	}

	// Calculate max width for each column
	maxLocalPath := len("Local Path")
	maxRemotePath := len("Remote Path")
	maxType := len("Type")
	maxStatus := len("Status")
	for _, s := range statuses {
		if len(s.LocalPath) > maxLocalPath {
			maxLocalPath = len(s.LocalPath)
		}
		if len(s.RemotePath) > maxRemotePath {
			maxRemotePath = len(s.RemotePath)
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

	// Print each status
	for _, s := range statuses {
		st := getStatusText(s)
		fmt.Printf("%-*s  %-*s  %-*s  %-*s\n", maxLocalPath, s.LocalPath, maxRemotePath, s.RemotePath, maxType, s.Type, maxStatus, st)
	}

	return nil
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

	// Get absolute path for remote directory
	absRemote, err := filepath.Abs(config.Remote)
	if err != nil {
		status.Error = fmt.Sprintf("Invalid remote directory path: %v", err)
		return status
	}

	// Set the full paths
	status.LocalPath = link.Path
	status.RemotePath = filepath.Join(absRemote, link.Path)

	// Check if the link path exists
	info, err := os.Stat(status.LocalPath)
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
		// Check if the file exists and is not a directory
		if info.IsDir() {
			status.Error = "Hard links cannot be created for directories"
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
