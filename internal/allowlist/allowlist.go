package allowlist

import (
	"bufio"
	"fmt"
	"os"
	"pa/internal/config"
	"strings"
)

// Checker answers whether a command is allowed for a given node.
// Patterns: one per line; leading/trailing whitespace and lines starting with # or empty are ignored.
// Matching: pattern ending with * is prefix match (command must start with pattern without *); otherwise exact match.
type Checker struct {
	nodePatterns map[string][]string // nodeID -> patterns (exact or prefix)
}

// NewChecker builds a checker from config. Loads each node's allowlist file (same path shared by multiple nodes is loaded once).
// All paths in config are relative to project root (CWD at startup); absolute paths are used as-is.
func NewChecker(cfg *config.Config) (*Checker, error) {
	pathToPatterns := make(map[string][]string)
	nodePatterns := make(map[string][]string)

	for nodeID, node := range cfg.Nodes {
		path := strings.TrimSpace(node.CommandAllowlistPath)
		if path == "" {
			continue
		}
		patterns, ok := pathToPatterns[path]
		if !ok {
			var err error
			patterns, err = loadPatterns(path)
			if err != nil {
				return nil, fmt.Errorf("node %s allowlist %s: %w", nodeID, path, err)
			}
			pathToPatterns[path] = patterns
		}
		nodePatterns[nodeID] = patterns
	}

	return &Checker{nodePatterns: nodePatterns}, nil
}

func loadPatterns(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

// Allow returns true if command is allowed for the given node.
// Returns false if node is unknown or command does not match any pattern.
func (c *Checker) Allow(nodeID, command string) bool {
	patterns, ok := c.nodePatterns[nodeID]
	if !ok {
		return false
	}
	cmd := strings.TrimSpace(command)
	for _, p := range patterns {
		if match(p, cmd) {
			return true
		}
	}
	return false
}

func match(pattern, command string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return prefix == "" || strings.HasPrefix(command, prefix)
	}
	return pattern == command
}
