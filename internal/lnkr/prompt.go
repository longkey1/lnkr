package lnkr

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// confirm asks a yes/no question on stdin and returns true only for an
// explicit yes. Any read error (e.g. closed stdin) is treated as no.
func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Println()
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
