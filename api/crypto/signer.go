package crypto

import (
	"errors"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// MaxSignerKeys is the most pairs one crypto.register_signer binding claims.
const MaxSignerKeys = 64

// ErrKeyClaimed refuses a crypto.register_signer set naming a pair another live
// binding holds. A signer reopening its binding can meet its own old binding
// here, until the node notices that binding closed.
var ErrKeyClaimed = errors.New("key already claimed by a signer")

// SignerKey is one claim of a crypto.register_signer binding: a complete public
// key and one signature scheme. No value of either field stands for more than
// one key or scheme.
type SignerKey struct {
	Key    *PublicKey
	Scheme astral.String8
}

func (SignerKey) ObjectType() string { return "mod.crypto.signer_key" }

func (k SignerKey) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&k).WriteTo(w)
}

func (k *SignerKey) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(k).ReadFrom(r)
}

// SignerKeys is the fixed key set of a crypto.register_signer binding, sent once
// as the first object on the channel.
type SignerKeys struct {
	Keys []*SignerKey
}

func (SignerKeys) ObjectType() string { return "mod.crypto.signer_keys" }

func (k SignerKeys) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&k).WriteTo(w)
}

func (k *SignerKeys) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(k).ReadFrom(r)
}

// why through Objectify: encoding/json writes a nil slice as null, and an empty list
// is [] on this wire (astral/slice_value.go).
func (k SignerKeys) MarshalJSON() ([]byte, error) { return astral.Objectify(&k).MarshalJSON() }

func (k *SignerKeys) UnmarshalJSON(b []byte) error { return astral.Objectify(k).UnmarshalJSON(b) }

// SignTextRequest asks a signer, on its binding, for one signature over Text.
type SignTextRequest struct {
	RequestID astral.Nonce
	Key       *PublicKey
	Scheme    astral.String8
	Text      astral.String16
}

func (SignTextRequest) ObjectType() string { return "mod.crypto.sign_text_request" }

func (r SignTextRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *SignTextRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// SignHashRequest asks a signer, on its binding, for one signature over Hash.
type SignHashRequest struct {
	RequestID astral.Nonce
	Key       *PublicKey
	Scheme    astral.String8
	Hash      Hash
}

func (SignHashRequest) ObjectType() string { return "mod.crypto.sign_hash_request" }

func (r SignHashRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *SignHashRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// SignAnswer is a signer's reply to one request, echoing RequestID. It carries
// exactly one of Signature and Error.
type SignAnswer struct {
	RequestID astral.Nonce
	Signature *Signature
	Error     astral.String16
}

func (SignAnswer) ObjectType() string { return "mod.crypto.sign_answer" }

func (a SignAnswer) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *SignAnswer) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// SignCancel tells a signer the node no longer waits for a request. It is
// best-effort: the node ignores an answer that arrives after it.
type SignCancel struct {
	RequestID astral.Nonce
}

func (SignCancel) ObjectType() string { return "mod.crypto.sign_cancel" }

func (c SignCancel) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&c).WriteTo(w)
}

func (c *SignCancel) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(c).ReadFrom(r)
}

func init() {
	astral.MustAdd(&SignerKey{})
	astral.MustAdd(&SignerKeys{})
	astral.MustAdd(&SignTextRequest{})
	astral.MustAdd(&SignHashRequest{})
	astral.MustAdd(&SignAnswer{})
	astral.MustAdd(&SignCancel{})
}
