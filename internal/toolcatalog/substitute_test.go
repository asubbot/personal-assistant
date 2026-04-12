package toolcatalog

import (
	"testing"
)

// Covers AC-04.007: valid tool call → arguments substituted into template → single command string.
func TestSubstitute_ReplacesPlaceholders(t *testing.T) {
	template := "systemctl status {{service}}"
	args := map[string]any{"service": "nginx"}
	out, err := Substitute(template, args)
	if err != nil {
		t.Fatalf("Substitute: %v", err)
	}
	if out != "systemctl status nginx" {
		t.Errorf("Substitute: got %q, want systemctl status nginx", out)
	}
}

// Covers AC-04.007: traceability for TestSubstitute_MissingPlaceholder_ReturnsError.
func TestSubstitute_MissingPlaceholder_ReturnsError(t *testing.T) {
	template := "cmd {{foo}} {{bar}}"
	args := map[string]any{"foo": "a"}
	_, err := Substitute(template, args)
	if err == nil {
		t.Fatal("Substitute(missing bar): expected error, got nil")
	}
	if err.Error() != `template: missing argument for placeholder "bar"` {
		t.Errorf("Substitute: error = %v", err)
	}
}

// Covers AC-04.007: traceability for TestSubstitute_NumberFormattedAsString.
func TestSubstitute_NumberFormattedAsString(t *testing.T) {
	template := "level {{n}}"
	args := map[string]any{"n": float64(42)}
	out, err := Substitute(template, args)
	if err != nil {
		t.Fatalf("Substitute: %v", err)
	}
	if out != "level 42" {
		t.Errorf("Substitute: got %q, want level 42", out)
	}
}

// Covers AC-04.007: traceability for TestSubstituteMust_PanicsOnMissing.
func TestSubstituteMust_PanicsOnMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("SubstituteMust(missing arg): expected panic")
		}
	}()
	SubstituteMust("{{x}}", map[string]any{})
}
