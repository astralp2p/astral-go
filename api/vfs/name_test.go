package vfs

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	valid := []string{"a", "a.txt", ".hidden", "...", "sp ace", "é", strings.Repeat("n", 255)}
	invalid := map[string]string{
		"empty":              "",
		"dot":                ".",
		"dot dot":            "..",
		"slash":              "a/b",
		"leading slash":      "/a",
		"NUL":                "a\x00b",
		"invalid UTF-8":      "a\xffb",
		"256 bytes":          strings.Repeat("n", 256),
		"128 two-byte runes": strings.Repeat("é", 128),
	}

	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("%q refused: %v", name, err)
		}
	}
	for label, name := range invalid {
		if err := ValidateName(name); err != ErrInvalidPath {
			t.Errorf("%s: got %v, want %v", label, err, ErrInvalidPath)
		}
	}
}
