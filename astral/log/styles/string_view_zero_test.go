package styles

import (
	"bytes"
	"testing"
)

// The zero value reaches the codec whenever a caller composes one; the constructor is
// not the only way in.
func TestStringView_ZeroValueDoesNotPanic(t *testing.T) {
	var v StringView

	var buf bytes.Buffer
	if _, err := v.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got StringView
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got.String() != "" {
		t.Errorf("want an empty string, got %q", got.String())
	}
}
