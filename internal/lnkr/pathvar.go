package lnkr

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Well-known variable names for ContractPath
const (
	VarHome           = "$HOME"
	VarLnkrRemoteRoot = "$LNKR_REMOTE_ROOT"
	VarPWD            = "$PWD"
)

// variablePattern matches $VARNAME or ${VARNAME} patterns
var variablePattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

// ExpandPath expands environment variables in a path string.
// Supports any environment variable in $VARNAME or ${VARNAME} format.
// $PWD is handled specially using os.Getwd().
// Returns error if a variable is undefined.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// If no $ in path, return as-is (backward compatibility with absolute paths)
	if !strings.Contains(path, "$") {
		return path, nil
	}

	result := path

	// Special handling for $PWD (not an environment variable on all systems)
	if strings.Contains(result, "$PWD") {
		pwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory for $PWD: %w", err)
		}
		result = strings.ReplaceAll(result, "${PWD}", pwd)
		result = strings.ReplaceAll(result, "$PWD", pwd)
	}

	// Find all variables and check if they are defined
	matches := variablePattern.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		varName := match[1]
		varValue := os.Getenv(varName)
		if varValue == "" {
			return "", fmt.Errorf("environment variable %q is not set", varName)
		}
	}

	// Expand all environment variables
	result = os.ExpandEnv(result)

	// Clean the path
	return filepath.Clean(result), nil
}

// ContractPath converts an absolute path to use variables where possible.
// Priority: $LNKR_REMOTE_ROOT > $HOME > $PWD (longer prefix wins)
func ContractPath(path string) string {
	if path == "" {
		return ""
	}

	// Don't contract if already contains variables
	if strings.Contains(path, "$") {
		return path
	}

	// Make sure path is absolute for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	type replacement struct {
		prefix   string
		variable string
	}

	var replacements []replacement

	// Collect possible replacements
	if remoteRoot := os.Getenv("LNKR_REMOTE_ROOT"); remoteRoot != "" {
		if absRemoteRoot, err := filepath.Abs(remoteRoot); err == nil {
			replacements = append(replacements, replacement{absRemoteRoot, VarLnkrRemoteRoot})
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		replacements = append(replacements, replacement{home, VarHome})
	}

	if pwd, err := os.Getwd(); err == nil {
		replacements = append(replacements, replacement{pwd, VarPWD})
	}

	// Find the best match (longest prefix)
	var bestMatch replacement
	for _, r := range replacements {
		if strings.HasPrefix(absPath, r.prefix) {
			if len(r.prefix) > len(bestMatch.prefix) {
				bestMatch = r
			}
		}
	}

	// Apply best match
	if bestMatch.prefix != "" {
		// Exact match
		if absPath == bestMatch.prefix {
			return bestMatch.variable
		}
		// Prefix match with separator
		if strings.HasPrefix(absPath, bestMatch.prefix+string(os.PathSeparator)) {
			suffix := absPath[len(bestMatch.prefix):]
			return bestMatch.variable + suffix
		}
	}

	return path
}
