package apps

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// A rejected objects.blueprints names the op and the action the app lacks; any
// other error passes unchanged.
func TestNameRefusal(t *testing.T) {
	err := nameRefusal(astral.NewErrRejected(0))
	want := "blueprint sync: objects.blueprints refused; the app needs mod.auth.use_blueprints_action"
	if !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("nameRefusal = %q; want prefix %q", err, want)
	}
	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("nameRefusal = %v; want it to wrap the rejection", err)
	}
	if got := nameRefusal(io.EOF); got != io.EOF {
		t.Fatalf("nameRefusal(io.EOF) = %v; want io.EOF", got)
	}
}
