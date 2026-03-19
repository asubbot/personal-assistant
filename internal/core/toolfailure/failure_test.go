package toolfailure

import (
	"errors"
	"fmt"
	"testing"
)

// Covers AC-06.004 (EP-006): policy-style tool errors do not qualify for escalation.
func TestQualifiesForEscalation_policyErrors(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{NoEscalate(errors.New(`tool catalog: unknown tool "x"`)), false},
		{NoEscalate(errors.New("noderunner: command not allowed for node \"n\"")), false},
		{NoEscalate(errors.New("noderunner: allowlist not configured")), false},
		{NoEscalate(fmt.Errorf("tool %q: %w", "t", errors.New(`command rejected: forbidden shell sequence ";" (REQ-04.031)`))), false},
		{MayEscalate(errors.New("noderunner: exec: remote failure")), true},
		{MayEscalate(errors.New("noderunner: ssh: connection refused")), true},
	}
	for _, tc := range cases {
		got := QualifiesForEscalation(tc.err)
		if got != tc.want {
			t.Errorf("QualifiesForEscalation(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

// Covers REQ-06.015: untyped errors do not qualify (fail closed).
func TestQualifiesForEscalation_untypedFailsClosed(t *testing.T) {
	if QualifiesForEscalation(errors.New("noderunner: exec: boom")) {
		t.Fatal("untyped error must not qualify")
	}
}
