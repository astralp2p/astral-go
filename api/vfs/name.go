package vfs

import (
	"strings"
	"unicode/utf8"
)

// MaxNameLen is the longest name in bytes.
const MaxNameLen = 255

// ValidateName reports a name that breaks the rules of VFS 6, section 3. The
// error is ErrInvalidPath.
//
// A name is valid UTF-8 of 1 to 255 bytes. It contains no '/' and no NUL byte,
// and it is not "." and not "..".
func ValidateName(name string) error {
	switch {
	case len(name) == 0, len(name) > MaxNameLen:
		return ErrInvalidPath
	case name == ".", name == "..":
		return ErrInvalidPath
	case strings.ContainsAny(name, "/\x00"):
		return ErrInvalidPath
	case !utf8.ValidString(name):
		return ErrInvalidPath
	}

	return nil
}
