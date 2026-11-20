package fmt

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

type testView string

func (v testView) Render() string { return string(v) }

func TestFprintWritesToWriter(t *testing.T) {
	var buf bytes.Buffer

	n, err := Fprint(&buf, testView("guest@host> "))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := buf.String(), "guest@host> "; got != want {
		t.Fatalf("writer holds %q, want %q", got, want)
	}

	if n != buf.Len() {
		t.Fatalf("n = %d, want %d", n, buf.Len())
	}
}

type hasViewProbe struct{ astral.String32 }

func (hasViewProbe) ObjectType() string { return "test.has_view_probe" }

// TestHasView records that a type has a view once one is registered for it.
func TestHasView(t *testing.T) {
	if HasView(&hasViewProbe{}) {
		t.Fatal("HasView is true before a view is registered")
	}

	SetView(func(o *hasViewProbe) View { return stringView("probe") })

	if !HasView(&hasViewProbe{}) {
		t.Error("HasView is false after a view is registered")
	}
}
