package id

import (
	"regexp"
	"strings"
	"testing"
)

// prefixedIDRe mirrors conformance/normalize.prefixedIDRe — generated IDs must
// match it so the conformance normalizer collapses them to <id>.
var prefixedIDRe = regexp.MustCompile(`^[a-z]+_[0-9A-Za-z]{20,}$`)

func TestFormat(t *testing.T) {
	for _, gen := range []struct {
		name string
		fn   func() string
	}{
		{"ascending", func() string { return Ascending(Session) }},
		{"descending", func() string { return Descending(Session) }},
	} {
		got := gen.fn()
		if !strings.HasPrefix(got, "ses_") {
			t.Errorf("%s: %q missing ses_ prefix", gen.name, got)
		}
		if !prefixedIDRe.MatchString(got) {
			t.Errorf("%s: %q does not match normalizer prefixedIDRe", gen.name, got)
		}
		// "<prefix>_" + 26 chars.
		if want := len("ses_") + length; len(got) != want {
			t.Errorf("%s: len(%q)=%d, want %d", gen.name, got, len(got), want)
		}
	}
}

func TestUnique(t *testing.T) {
	seen := make(map[string]bool, 10000)
	for i := 0; i < 10000; i++ {
		v := Descending(Session)
		if seen[v] {
			t.Fatalf("duplicate id generated: %q", v)
		}
		seen[v] = true
	}
}

func TestAscendingSortsChronologically(t *testing.T) {
	a := create(string(Session), false, 1000)
	b := create(string(Session), false, 2000)
	if a >= b {
		t.Errorf("ascending IDs not ordered: %q >= %q", a, b)
	}
}

func TestDescendingSortsNewestFirst(t *testing.T) {
	a := create(string(Session), true, 1000)
	b := create(string(Session), true, 2000)
	// Later timestamp should sort lexicographically BEFORE the earlier one.
	if b >= a {
		t.Errorf("descending IDs not newest-first: %q >= %q", b, a)
	}
}
