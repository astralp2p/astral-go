package ether

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// EnvelopeVersion is the Envelope version this module writes. A receiver drops a
// datagram whose first byte differs from it.
const EnvelopeVersion = 1

// Envelope is one ether datagram: a version byte, then one object as an interface slot
// (string8 type tag, then payload). A datagram is exactly the bytes WriteTo writes, with
// no outer type tag. A receiver drops a datagram with bytes after the Envelope.
//
// why: an old mod.ether.signed_broadcast datagram starts with the high byte of a
// UnixNano timestamp (0x17-0x1c for 2023-2034), so the version byte rejects it.
type Envelope struct {
	Version astral.Uint8
	Object  astral.Object
}

var _ astral.Object = &Envelope{}

// astral

func (Envelope) ObjectType() string { return "mod.ether.envelope" }

// why: Objectify decodes the slot through the caller's reader, so the depth and byte
// budgets and the Blueprints registry of the call carry into Object. A nested Decode
// over a fresh reader would reset them.
func (e Envelope) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *Envelope) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(e).ReadFrom(r)
}

// json

// MarshalJSON writes Object as a nested Type/Object container, the form of an
// interface-typed slot (.ai/system/topics/json-encoding.md).
func (e Envelope) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *Envelope) UnmarshalJSON(data []byte) error {
	return astral.Objectify(e).UnmarshalJSON(data)
}

func init() {
	_ = astral.Add(&Envelope{})
}
