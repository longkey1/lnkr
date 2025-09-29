package lnkr

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func Remove(path string) error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	var newLinks []Link
	removed := false
	for _, link := range config.Links {
		if link.Path == path || strings.HasPrefix(link.Path, path+string(os.PathSeparator)) {
			fmt.Printf("Removed link: %s\n", link.Path)
			removed = true
			continue
		}
		newLinks = append(newLinks, link)
	}

	if !removed {
		fmt.Println("No matching links found to remove.")
		return nil
	}

	// pathで昇順ソート
	sort.Slice(newLinks, func(i, j int) bool {
		return newLinks[i].Path < newLinks[j].Path
	})

	config.Links = newLinks
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Apply all remaining link paths to GitExclude
	if err := applyAllLinksToGitExclude(config); err != nil {
		fmt.Printf("Warning: failed to apply link paths to GitExclude: %v\n", err)
	}

	return nil
}
