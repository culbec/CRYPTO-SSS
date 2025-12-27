package strings

import (
    "testing"
    "regexp"
)

func TestStringsEcho(t *testing.T) {
    s := "Hello"
    want := regexp.MustCompile(`\b` + s + `\b`)
    
    echo := Echo(s)
    if !want.MatchString(echo) {
        t.Errorf(`Echo(%q) = %q, want match for %#q`, s, echo, want)
    }
}

func TestStringsReverse(t *testing.T) {
    s := "Hello"
    want := "olleH"
    
    rev := Reverse(s)
    if rev != want {
        t.Errorf(`Reverse(%q) = %q, want match for %#q`, s, rev, want)
    }
}
