// Package cmdsafe provides pre-execution checks for commands run on nodes (REQ-04.031):
// allowed rune set and UTF-8 validation (RejectDisallowedRunes), shell metacharacter rejection (RejectShellMetacharacters),
// and ValidateRemoteCommand as the single ordered gate for remote execution paths.
package cmdsafe

import (
	"fmt"
	"strings"
)

// forbiddenSubstrings is the minimum set from REQ-04.031 (semicolon, ampersand, pipe, newlines,
// dollar-parenthesis substitution open, backtick).
var forbiddenSubstrings = []string{";", "&", "|", "\n", "\r", "$(", "`"}

// RejectShellMetacharacters returns an error if cmd contains a forbidden sequence.
// Call before SSH exec / run_on_node for catalog-substituted commands and scheduler-supplied commands.
func RejectShellMetacharacters(cmd string) error {
	for _, s := range forbiddenSubstrings {
		if strings.Contains(cmd, s) {
			return fmt.Errorf("command rejected: forbidden shell sequence %q (REQ-04.031)", s)
		}
	}
	return nil
}
