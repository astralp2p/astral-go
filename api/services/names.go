package services

import (
	"errors"
	"fmt"
	"strings"
)

// MaxNames bounds the services one advertise or discover call names.
const MaxNames = 64

// maxNameLen is the longest name a string8 holds.
const maxNameLen = 255

var ErrNoNames = errors.New("no service names")

// ParseNames splits a comma-separated service list and validates it. A name is
// non-empty, holds no comma, has no leading or trailing whitespace, and fits a
// string8. A list holds at least one name, at most MaxNames, and no name twice.
func ParseNames(list string) ([]string, error) {
	if list == "" {
		return nil, ErrNoNames
	}

	names := strings.Split(list, ",")
	if len(names) > MaxNames {
		return nil, fmt.Errorf("more than %d service names", MaxNames)
	}

	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if err := ValidateName(name); err != nil {
			return nil, err
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("service %q named twice", name)
		}
		seen[name] = struct{}{}
	}

	return names, nil
}

// JoinNames is the inverse of ParseNames for valid names.
func JoinNames(names []string) string {
	return strings.Join(names, ",")
}

// ValidateName reports why name is not a valid service name, or nil.
func ValidateName(name string) error {
	switch {
	case name == "":
		return errors.New("empty service name")
	case len(name) > maxNameLen:
		return fmt.Errorf("service name longer than %d bytes", maxNameLen)
	case strings.Contains(name, ","):
		return fmt.Errorf("service name %q contains a comma", name)
	case strings.TrimSpace(name) != name:
		return fmt.Errorf("service name %q has surrounding whitespace", name)
	}
	return nil
}
