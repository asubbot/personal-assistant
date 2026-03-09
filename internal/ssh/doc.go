// Package ssh implements the SSH client for node access.
// All connections to a node must use only the dedicated user for that node (see DedicatedUser).
package ssh

import (
	"fmt"
	"pa/internal/config"
)

// DedicatedUser returns the single SSH user identity for the given node.
// The SSH client must use only this identity for that node; no shared or alternate account.
// Returns error if nodeID is not in config.
func DedicatedUser(cfg *config.Config, nodeID string) (string, error) {
	node, ok := cfg.Nodes[nodeID]
	if !ok {
		return "", fmt.Errorf("node %q not found in config", nodeID)
	}
	user := node.DedicatedUser
	if user == "" {
		return "", fmt.Errorf("node %q has no dedicated_user", nodeID)
	}
	return user, nil
}
