package test

import (
	"regexp"
	"testing"

	"github.com/culbec/CRYPTO-sss/src/backend/pkg/strings"
)

func TestStringsEcho(t *testing.T) {
	s := "Hello"
	want := regexp.MustCompile(`\b` + s + `\b`)

	echo := strings.Echo(s)
	if !want.MatchString(echo) {
		t.Errorf(`Echo(%q) = %q, want match for %#q`, s, echo, want)
	}
}

func TestStringsReverse(t *testing.T) {
	s := "Hello"
	want := "olleH"

	rev := strings.Reverse(s)
	if rev != want {
		t.Errorf(`Reverse(%q) = %q, want match for %#q`, s, rev, want)
	}
}
