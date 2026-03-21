package cmdsafe

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// maxCommandRunes caps command length after trim (DoS bound for SSH argument string).
const maxCommandRunes = 8192

// allowedASCIIPunct is the fixed set of ASCII punctuation allowed in addition to letters,
// numbers, and combining marks (Mn, Mc). Space is U+0020 only; tab is not allowed.
var allowedASCIIPunct = map[rune]struct{}{
	' ': {}, '.': {}, '/': {}, '-': {}, '_': {}, ':': {}, '@': {}, '=': {}, '+': {}, ',': {},
}

func isBidiOverride(r rune) bool {
	switch r {
	case '\u202a', '\u202b', '\u202c', '\u202d', '\u202e',
		'\u2066', '\u2067', '\u2068', '\u2069':
		return true
	default:
		return false
	}
}

func runeAllowed(r rune) bool {
	if isBidiOverride(r) {
		return false
	}
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return true
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) {
		return true
	}
	_, ok := allowedASCIIPunct[r]
	return ok
}

// RejectDisallowedRunes returns an error if cmd is not valid UTF-8, exceeds maxCommandRunes,
// or contains a rune outside the allowed set: Unicode letters, numbers, Mn/Mc marks, and
// allowedASCIIPunct. Control (Cc), format (Cf), surrogate (Cs), private-use (Co), and
// bidi override characters are rejected. Call before RejectShellMetacharacters.
func RejectDisallowedRunes(cmd string) error {
	if !utf8.ValidString(cmd) {
		return fmt.Errorf("command rejected: invalid UTF-8")
	}
	if utf8.RuneCountInString(cmd) > maxCommandRunes {
		return fmt.Errorf("command rejected: exceeds max length (%d runes)", maxCommandRunes)
	}
	const charSetHint = " (not in allowed command character set)"
	for _, r := range cmd {
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cs, r) || unicode.Is(unicode.Co, r) {
			return fmt.Errorf("command rejected: disallowed character U+%04X%s", r, charSetHint)
		}
		if !runeAllowed(r) {
			return fmt.Errorf("command rejected: disallowed character U+%04X%s", r, charSetHint)
		}
	}
	return nil
}
