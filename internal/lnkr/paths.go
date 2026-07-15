package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveLocalRelPath converts an input path (absolute, or relative to the
// current working directory) into a path relative to localDir.
// A relative path that does not point inside localDir from the current
// directory is interpreted as relative to localDir itself, so paths stored in
// the configuration keep working even when the command runs outside localDir.
func resolveLocalRelPath(input, localDir string) (string, error) {
	cleaned := filepath.Clean(input)

	var candidates []string
	if filepath.IsAbs(cleaned) {
		candidates = []string{cleaned}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		candidates = []string{filepath.Join(cwd, cleaned), filepath.Join(localDir, cleaned)}
	}

	var firstMatch string
	for _, candidate := range candidates {
		rel, ok := relPathWithin(localDir, candidate)
		if !ok {
			continue
		}
		if _, err := os.Lstat(filepath.Join(localDir, rel)); err == nil {
			return rel, nil
		}
		if firstMatch == "" {
			firstMatch = rel
		}
	}
	if firstMatch != "" {
		return firstMatch, nil
	}
	return "", fmt.Errorf("path is outside the local directory (%s): %s", localDir, input)
}

// relPathWithin returns target relative to baseDir when target is strictly
// inside baseDir. Symlinked path prefixes (e.g. /var vs /private/var on
// macOS) are resolved before giving up.
func relPathWithin(baseDir, target string) (string, bool) {
	if rel, ok := tryRel(baseDir, target); ok {
		return rel, true
	}

	resolvedBase, errBase := filepath.EvalSymlinks(baseDir)
	dir, file := filepath.Split(filepath.Clean(target))
	resolvedDir, errDir := filepath.EvalSymlinks(filepath.Clean(dir))
	if errBase != nil || errDir != nil {
		return "", false
	}
	return tryRel(resolvedBase, filepath.Join(resolvedDir, file))
}

func tryRel(baseDir, target string) (string, bool) {
	rel, err := filepath.Rel(baseDir, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return rel, true
}
