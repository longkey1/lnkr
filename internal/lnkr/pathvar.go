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
	VarLnkrLocalRoot  = "$LNKR_LOCAL_ROOT"
	VarPWD            = "$PWD"
)

// variablePattern matches $VARNAME or ${VARNAME} patterns
var variablePattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

// placeholderPattern matches {{key}} patterns for global config references (env > config > default)
var placeholderPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ExpandPath expands placeholders and environment variables in a path string.
// Supports:
// - Placeholders: {{remote_root}}, {{local_root}} (env > config > default priority)
// - Environment variables: $VARNAME or ${VARNAME} format
// - Special: $PWD (current working directory)
// Returns error if a variable/placeholder is undefined.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	result := path

	// Expand placeholders {{key}} first
	if strings.Contains(result, "{{") {
		result = expandPlaceholders(result)
	}

	// If no $ in path, return as-is (backward compatibility with absolute paths)
	if !strings.Contains(result, "$") {
		return filepath.Clean(result), nil
	}

	// Special handling for $PWD (not an environment variable on all systems)
	if strings.Contains(result, "$PWD") {
		pwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory for $PWD: %w", err)
		}
		result = strings.ReplaceAll(result, "${PWD}", pwd)
		result = strings.ReplaceAll(result, "$PWD", pwd)
	}

	// Special handling for LNKR variables - use global config if env var not set
	if strings.Contains(result, "$LNKR_REMOTE_ROOT") || strings.Contains(result, "${LNKR_REMOTE_ROOT}") {
		value := GetRemoteRoot() // This already handles env > config > default priority
		result = strings.ReplaceAll(result, "${LNKR_REMOTE_ROOT}", value)
		result = strings.ReplaceAll(result, "$LNKR_REMOTE_ROOT", value)
	}

	if strings.Contains(result, "$LNKR_LOCAL_ROOT") || strings.Contains(result, "${LNKR_LOCAL_ROOT}") {
		value := GetLocalRoot() // This already handles env > config priority
		if value == "" {
			return "", fmt.Errorf("LNKR_LOCAL_ROOT is not set in environment or config file")
		}
		result = strings.ReplaceAll(result, "${LNKR_LOCAL_ROOT}", value)
		result = strings.ReplaceAll(result, "$LNKR_LOCAL_ROOT", value)
	}

	// Find all remaining variables and check if they are defined
	matches := variablePattern.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		varName := match[1]
		varValue := os.Getenv(varName)
		if varValue == "" {
			return "", fmt.Errorf("environment variable %q is not set", varName)
		}
	}

	// Expand all remaining environment variables
	result = os.ExpandEnv(result)

	// Clean the path
	return filepath.Clean(result), nil
}

// expandPlaceholders replaces {{key}} with values (env > config > default priority)
func expandPlaceholders(path string) string {
	return placeholderPattern.ReplaceAllStringFunc(path, func(match string) string {
		// Extract key from {{key}}
		submatch := placeholderPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		key := submatch[1]
		switch key {
		case "remote_root":
			return GetRemoteRoot()
		case "local_root":
			return GetLocalRoot()
		case "link_type":
			return GetGlobalLinkType()
		case "git_exclude_path":
			return GetGlobalGitExcludePath()
		default:
			return match // Keep unknown placeholders as-is
		}
	})
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

	// Collect possible replacements (use global config which handles env > config priority)
	if remoteRoot := GetRemoteRoot(); remoteRoot != "" {
		if absRemoteRoot, err := filepath.Abs(remoteRoot); err == nil {
			replacements = append(replacements, replacement{absRemoteRoot, VarLnkrRemoteRoot})
		}
	}

	if localRoot := GetLocalRoot(); localRoot != "" {
		if absLocalRoot, err := filepath.Abs(localRoot); err == nil {
			replacements = append(replacements, replacement{absLocalRoot, VarLnkrLocalRoot})
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
