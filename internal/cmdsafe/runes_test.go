package cmdsafe

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// REQ-04.031 / AC-04.029 (with shellmeta_test.go, remote_test.go): command character and shell policy before node exec.
func TestRejectDisallowedRunes_allowed(t *testing.T) {
	cases := []string{
		"uptime",
		"/usr/bin/rsync -avz /src /dst",
		"echo hello",
		"/volume1/homes/x/.local/bin/sonos Kitchen volume 30",
		"/opt/привет/world",
		"Кириллица команда",
		"cafe\u0301",
		"a-z_A-Z0-9./:@=+, ",
		`docker run --rm --network bridge pa-sandbox:python timeout 30s echo ok`,
		`echo "hello world"`,
	}
	for _, cmd := range cases {
		if err := RejectDisallowedRunes(cmd); err != nil {
			t.Errorf("RejectDisallowedRunes(%q): %v", cmd, err)
		}
	}
}

// Covers AC-04.029: traceability for TestRejectDisallowedRunes_rejectsControlAndTab.
func TestRejectDisallowedRunes_rejectsControlAndTab(t *testing.T) {
	for _, cmd := range []string{
		"uptime\x00",
		"a\tb",
		"\x1b[0m",
	} {
		err := RejectDisallowedRunes(cmd)
		if err == nil {
			t.Fatalf("RejectDisallowedRunes(%q): want error", cmd)
		}
		if !strings.Contains(err.Error(), "disallowed character") && !strings.Contains(err.Error(), "invalid UTF-8") {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(err.Error(), "not in allowed command character set") {
			t.Errorf("want character-set hint in error: %v", err)
		}
	}
}

// Covers AC-04.029: traceability for TestRejectDisallowedRunes_rejectsFormatAndBidi.
func TestRejectDisallowedRunes_rejectsFormatAndBidi(t *testing.T) {
	for _, cmd := range []string{
		"foo\u200bbar",
		"left\u202eright",
		"x\u2066y",
	} {
		err := RejectDisallowedRunes(cmd)
		if err == nil {
			t.Fatalf("RejectDisallowedRunes(%q): want error", cmd)
		}
	}
}

// Covers AC-04.029: traceability for TestRejectDisallowedRunes_rejectsInvalidUTF8.
func TestRejectDisallowedRunes_rejectsInvalidUTF8(t *testing.T) {
	cmd := "ok\xff\xfe"
	err := RejectDisallowedRunes(cmd)
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("got err=%v, want invalid UTF-8", err)
	}
}

// Covers AC-04.029: traceability for TestRejectDisallowedRunes_rejectsDisallowedSymbol.
func TestRejectDisallowedRunes_rejectsDisallowedSymbol(t *testing.T) {
	err := RejectDisallowedRunes("echo 😀")
	if err == nil {
		t.Fatal("want error for emoji")
	}
	if !strings.Contains(err.Error(), "disallowed character") || !strings.Contains(err.Error(), "not in allowed command character set") {
		t.Errorf("got %v", err)
	}
}

// Covers AC-04.029: traceability for TestRejectDisallowedRunes_rejectsTilde.
func TestRejectDisallowedRunes_rejectsTilde(t *testing.T) {
	err := RejectDisallowedRunes("uptime~")
	if err == nil {
		t.Fatal("want error: ~ not in allowed punctuation")
	}
	if !strings.Contains(err.Error(), "not in allowed command character set") {
		t.Errorf("want hint: %v", err)
	}
}

// NBSP (U+00A0) is not the allowed ASCII space U+0020; reject to avoid accidentally widening Zs.
// Covers AC-04.029: traceability for TestRejectDisallowedRunes_rejectsNBSP.
func TestRejectDisallowedRunes_rejectsNBSP(t *testing.T) {
	cmd := "uptime\u00a0"
	err := RejectDisallowedRunes(cmd)
	if err == nil {
		t.Fatal("want error for no-break space")
	}
	if !strings.Contains(err.Error(), "disallowed character") || !strings.Contains(err.Error(), "00A0") {
		t.Fatalf("got %v, want disallowed U+00A0", err)
	}
	if !strings.Contains(err.Error(), "not in allowed command character set") {
		t.Errorf("want character-set hint: %v", err)
	}
}

// Covers AC-04.029: traceability for TestRejectDisallowedRunes_maxLength.
func TestRejectDisallowedRunes_maxLength(t *testing.T) {
	var sb strings.Builder
	for utf8.RuneCountInString(sb.String()) < maxCommandRunes {
		sb.WriteRune('a')
	}
	if err := RejectDisallowedRunes(sb.String()); err != nil {
		t.Fatalf("exactly max runes: %v", err)
	}
	sb.WriteRune('a')
	overErr := RejectDisallowedRunes(sb.String())
	if overErr == nil {
		t.Fatal("want error over max runes")
	}
	if !strings.Contains(overErr.Error(), "max length") {
		t.Errorf("got %v", overErr)
	}
}
