package ether

import (
	"io"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

var _ astral.Object = &SignedBroadcast{}

type SignedBroadcast struct {
	Broadcast
	Signature *crypto.Signature
}

func (SignedBroadcast) ObjectType() string {
	return "mod.ether.signed_broadcast"
}

func (b SignedBroadcast) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&b).WriteTo(w)
}

func (b *SignedBroadcast) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(b).ReadFrom(r)
}

// json

// signedBroadcastJSON is the JSON payload of SignedBroadcast: the embedded Broadcast's
// fields sit flat beside Signature, as encoding/json writes an embedded struct.
//
// why: Objectify writes an embedded struct under its field name, and without its own
// methods SignedBroadcast would inherit Broadcast's and drop Signature.
type signedBroadcastJSON struct {
	Timestamp astral.Time
	Source    *astral.Identity
	Object    astral.Object
	Signature *crypto.Signature
}

// MarshalJSON writes Object as a nested Type/Object container, the form of an
// interface-typed slot (.ai/system/topics/json-encoding.md).
func (b SignedBroadcast) MarshalJSON() ([]byte, error) {
	j := signedBroadcastJSON{
		Timestamp: b.Timestamp,
		Source:    b.Source,
		Object:    b.Object,
		Signature: b.Signature,
	}
	return astral.Objectify(&j).MarshalJSON()
}

func (b *SignedBroadcast) UnmarshalJSON(data []byte) error {
	var j signedBroadcastJSON
	err := astral.Objectify(&j).UnmarshalJSON(data)
	if err != nil {
		return err
	}

	b.Broadcast = Broadcast{Timestamp: j.Timestamp, Source: j.Source, Object: j.Object}
	b.Signature = j.Signature
	return nil
}

// Hash returns the object ID hash of the embedded Broadcast, not the full SignedBroadcast,
// so the signature covers only the payload and not the signature field itself.
func (b SignedBroadcast) Hash() []byte {
	objectID, err := astral.ResolveObjectID(&b.Broadcast)
	if err != nil {
		return nil
	}
	return objectID.Hash[:]
}

func init() {
	_ = astral.Add(&SignedBroadcast{})
}
