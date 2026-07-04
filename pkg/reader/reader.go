package reader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ReadKeywords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	seen := make(map[string]bool)
	var kws []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !seen[line] {
			seen[line] = true
			kws = append(kws, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	if len(kws) == 0 {
		return nil, fmt.Errorf("no keywords found in %s", path)
	}
	return kws, nil
}
