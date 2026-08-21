package pub_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	_ "github.com/astralp2p/astral-go"
	"github.com/astralp2p/astral-go/astral"
)

// WriteTo and ReadFrom report how many bytes they moved, and nothing in the module
// checks that claim against the stream. Six of the ten scalars disagreed with it, two
// accumulator sites dropped bytes they had consumed, and one type encoder disagreed
// with its own decoder — none of which any test noticed. Counting what actually
// crosses the wire is the only thing that catches this class.
//
// This lives at the repo root because it needs the blank import above to register the
// whole api surface, and astral cannot import the root package.

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// auditCounts writes o, reads it back into fresh, and compares each side's
// self-reported count against the bytes that actually moved.
func auditCounts(o, fresh astral.Object) (problems []string) {
	defer func() {
		if r := recover(); r != nil {
			problems = append(problems, fmt.Sprintf("panic: %v", r))
		}
	}()

	var buf bytes.Buffer
	cw := &countingWriter{w: &buf}

	reported, err := o.WriteTo(cw)
	if err != nil {
		return nil // not every zero value is encodable; skip rather than fail
	}
	if reported != cw.n {
		problems = append(problems, fmt.Sprintf("WriteTo reported %d, wrote %d", reported, cw.n))
	}

	cr := &countingReader{r: bytes.NewReader(buf.Bytes())}

	reread, err := fresh.ReadFrom(cr)
	if err != nil {
		return problems
	}
	if reread != cr.n {
		problems = append(problems, fmt.Sprintf("ReadFrom reported %d, read %d", reread, cr.n))
	}

	return problems
}

func TestRegisteredPrototypeCounts(t *testing.T) {
	for _, name := range astral.DefaultBlueprints().OrderedBlueprints() {
		t.Run(name, func(t *testing.T) {
			o, fresh := astral.New(name), astral.New(name)
			if o == nil || fresh == nil {
				t.Skipf("%s does not materialize", name)
			}
			for _, p := range auditCounts(o, fresh) {
				t.Errorf("%s: %s", name, p)
			}
		})
	}
}

// sliceValue and mapValue are unexported and unregistered, so the sweep above cannot
// reach them. A local struct with container fields is the only way in.
type countHolder struct {
	Items  []astral.Uint8
	Lookup map[astral.String8]astral.Uint16
	Fixed  [3]astral.Uint32
	Scalar astral.Uint64
}

func TestObjectifyCompositeCounts(t *testing.T) {
	cases := map[string]countHolder{
		"populated": {
			Items:  []astral.Uint8{1, 2, 3},
			Lookup: map[astral.String8]astral.Uint16{"a": 1},
			Fixed:  [3]astral.Uint32{4, 5, 6},
			Scalar: 7,
		},
		// the length prefix is read either way, so an empty slice exercises the
		// accumulator without any elements to mask it
		"empty": {Items: []astral.Uint8{}, Lookup: map[astral.String8]astral.Uint16{}},
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			var fresh countHolder
			for _, p := range auditCounts(astral.Objectify(&value), astral.Objectify(&fresh)) {
				t.Errorf("%s", p)
			}
		})
	}
}

// The type encoders are plain functions, not WriteTo methods, so no method-shaped
// audit reaches them. Encoder and decoder must also agree with each other.
func TestTypeEncoderCounts(t *testing.T) {
	encoders := map[string]struct {
		encode func(io.Writer, string) (int64, error)
		decode func(io.Reader) (string, int64, error)
	}{
		"canonical": {astral.CanonicalTypeEncoder, astral.CanonicalTypeDecoder},
		"short":     {astral.ShortTypeEncoder, astral.ShortTypeDecoder},
	}

	for name, codec := range encoders {
		t.Run(name, func(t *testing.T) {
			const typeName = "uint8"

			var buf bytes.Buffer
			cw := &countingWriter{w: &buf}

			reported, err := codec.encode(cw, typeName)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if reported != cw.n {
				t.Errorf("encoder reported %d, wrote %d", reported, cw.n)
			}

			cr := &countingReader{r: bytes.NewReader(buf.Bytes())}

			got, reread, err := codec.decode(cr)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != typeName {
				t.Errorf("round-trip: want %q, got %q", typeName, got)
			}
			if reread != cr.n {
				t.Errorf("decoder reported %d, read %d", reread, cr.n)
			}
			if reported != reread {
				t.Errorf("encoder reported %d but decoder reported %d for the same stream", reported, reread)
			}
		})
	}
}
