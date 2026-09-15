package fmt

import (
	"bytes"
	"testing"
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
