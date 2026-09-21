package objects

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// sampleProbes covers a probe without ObjectID, as a node that predates the field
// answers, a probe that resolves a stored object, and a probe that resolves the empty
// object, whose full ID has Size 0.
func sampleProbes(t *testing.T) map[string]Probe {
	t.Helper()

	full, err := astral.ParseID("data1rxqff36hhoddbhwbsd5c1smbpoh9oq5pgum6n6g4bg1esia4psp1r")
	if err != nil {
		t.Fatal(err)
	}
	empty := &astral.ObjectID{Hash: sha256.Sum256(nil)}

	return map[string]Probe{
		"nil ObjectID": {
			Type: "string8", Repo: "local", Mime: "text/plain; charset=utf-8", Time: 421000,
		},
		"full ObjectID": {
			Type: "string8", Repo: "local", Mime: "text/plain; charset=utf-8", Time: 421000,
			ObjectID: full,
		},
		"empty object": {
			Repo: "mem0", Mime: "text/plain; charset=utf-8", Time: 1200,
			ObjectID: empty,
		},
	}
}

func TestProbe_BinaryRoundTrip(t *testing.T) {
	for name, src := range sampleProbes(t) {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := src.WriteTo(&buf); err != nil {
				t.Fatal(err)
			}

			var dst Probe
			if _, err := dst.ReadFrom(&buf); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(src, dst) {
				t.Fatalf("want %+v, got %+v", src, dst)
			}
		})
	}
}

func TestProbe_JSONRoundTrip(t *testing.T) {
	for name, src := range sampleProbes(t) {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(src)
			if err != nil {
				t.Fatal(err)
			}

			var dst Probe
			if err := json.Unmarshal(data, &dst); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(src, dst) {
				t.Fatalf("want %+v, got %+v from %s", src, dst, data)
			}
		})
	}
}

// The objects.probe shape: keys in alphabetical order, ObjectID in its data1 form, and
// null for a nil ObjectID.
func TestProbe_MarshalJSON_Shape(t *testing.T) {
	probes := sampleProbes(t)
	cases := map[string]string{
		"nil ObjectID":  `{"Mime":"text/plain; charset=utf-8","ObjectID":null,"Repo":"local","Time":421000,"Type":"string8"}`,
		"full ObjectID": `{"Mime":"text/plain; charset=utf-8","ObjectID":"data1rxqff36hhoddbhwbsd5c1smbpoh9oq5pgum6n6g4bg1esia4psp1r","Repo":"local","Time":421000,"Type":"string8"}`,
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(probes[name])
			if err != nil {
				t.Fatal(err)
			}

			if string(data) != want {
				t.Fatalf("want %s, got %s", want, data)
			}
		})
	}
}

// probeWithoutObjectID is Probe before it carried ObjectID.
type probeWithoutObjectID struct {
	Type astral.String8
	Repo astral.String8
	Mime astral.String8
	Time astral.Duration
}

// A payload from a node that predates ObjectID ends before the field's presence byte.
// Decoding it fails with io.EOF; the break is accepted.
func TestProbe_ReadFrom_PreChangePayloadIsEOF(t *testing.T) {
	old := probeWithoutObjectID{Type: "string8", Repo: "local", Mime: "text/plain; charset=utf-8", Time: 421000}

	var buf bytes.Buffer
	if _, err := astral.Objectify(&old).WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	var dst Probe
	_, err := dst.ReadFrom(&buf)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("want io.EOF, got %v", err)
	}
}

// ObjectID follows Time on the wire: a presence byte of 0 for nil, or 1 and the 40-byte
// ID.
func TestProbe_WriteTo_AppendsObjectID(t *testing.T) {
	probes := sampleProbes(t)

	old := probeWithoutObjectID{Type: "string8", Repo: "local", Mime: "text/plain; charset=utf-8", Time: 421000}
	var prefix bytes.Buffer
	if _, err := astral.Objectify(&old).WriteTo(&prefix); err != nil {
		t.Fatal(err)
	}
	var id bytes.Buffer
	if _, err := probes["full ObjectID"].ObjectID.WriteTo(&id); err != nil {
		t.Fatal(err)
	}

	cases := map[string][]byte{
		"nil ObjectID":  slices.Concat(prefix.Bytes(), []byte{0}),
		"full ObjectID": slices.Concat(prefix.Bytes(), []byte{1}, id.Bytes()),
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			var got bytes.Buffer
			if _, err := probes[name].WriteTo(&got); err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(got.Bytes(), want) {
				t.Fatalf("want % x, got % x", want, got.Bytes())
			}
		})
	}
}
