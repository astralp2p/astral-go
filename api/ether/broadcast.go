package ether

import (
	"bytes"
	"errors"
	"io"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/streams"
)

var _ astral.Object = &Broadcast{}

type Broadcast struct {
	Timestamp astral.Time
	Source    *astral.Identity
	Object    astral.Object
}

var _ astral.Object = &Broadcast{}

// astral

func (Broadcast) ObjectType() string { return "mod.ether.broadcast" }

// DeclareBlueprint describes the bytes WriteTo writes. Source is the bare 33-byte identity,
// with no presence flag. Object is a bytes16 container holding the payload's type tag and
// bytes.
func (Broadcast) DeclareBlueprint() *astral.Blueprint {
	return astral.NewBlueprint(Broadcast{}.ObjectType(),
		astral.Field{Name: "Timestamp", Spec: &astral.PrimitiveSpec{PrimitiveType: "time"}},
		astral.Field{Name: "Source", Spec: &astral.PrimitiveSpec{PrimitiveType: "identity"}},
		astral.Field{Name: "Object", Spec: &astral.PrimitiveSpec{PrimitiveType: "bytes16"}},
	)
}

func (b Broadcast) WriteTo(w io.Writer) (n int64, err error) {
	if b.Source == nil {
		return 0, errors.New("nil source")
	}
	if b.Object == nil {
		return 0, errors.New("nil object")
	}

	var buf = &bytes.Buffer{}
	if _, err = astral.Encode(buf, b.Object); err != nil {
		return 0, err
	}

	return streams.WriteAllTo(w,
		b.Timestamp,
		b.Source,
		astral.Bytes16(buf.Bytes()),
	)
}

func (b *Broadcast) ReadFrom(r io.Reader) (n int64, err error) {
	var buf astral.Bytes16
	b.Source = &astral.Identity{}

	n, err = streams.ReadAllFrom(r, &b.Timestamp, b.Source, &buf)
	if err != nil {
		return
	}

	b.Object, _, err = astral.Decode(bytes.NewReader(buf))
	if err != nil {
		return
	}

	return
}

// json

// MarshalJSON writes Object as a nested Type/Object container, the form of an
// interface-typed slot (.ai/system/topics/json-encoding.md).
func (b Broadcast) MarshalJSON() ([]byte, error)     { return astral.Objectify(&b).MarshalJSON() }
func (b *Broadcast) UnmarshalJSON(data []byte) error { return astral.Objectify(b).UnmarshalJSON(data) }

func init() {
	_ = astral.Add(&Broadcast{})
}
