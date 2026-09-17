package tor

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"errors"
	"github.com/astralp2p/astral-go/astral"
	"io"
	"strings"
)

const DigestSize = 35

// ErrInvalidDigestLength indicates the digest is not DigestSize bytes long.
var ErrInvalidDigestLength = errors.New("invalid digest length")

// zeroDigest is the wire form of a digest that names no onion service.
var zeroDigest [DigestSize]byte

// zeroDigestText is the text form of a digest that names no onion service. It is
// the string mod.tor.endpoint uses for the same absence.
const zeroDigestText = "unknown"

// Digest is an astral.Object that holds a Tor digest. Supports JSON and text.
type Digest []byte

var _ astral.Object = (*Digest)(nil)

// DigestFromString parses a base32-encoded Tor digest, with or without the ".onion" suffix.
func DigestFromString(s string) (Digest, error) {
	var d = Digest{}
	err := d.UnmarshalText([]byte(s))
	return d, err
}

// astral.Object

func (d Digest) ObjectType() string { return "mod.tor.digest" }

// WriteTo writes the digest as DigestSize raw bytes, whatever it holds. The zero
// value writes DigestSize null bytes, which ReadFrom reads back as the zero value;
// a digest of any other length is not a digest and does not reach the wire.
//
// The width cannot depend on the value: ReadFrom demands DigestSize, so a shorter
// write shifts every field after it in the enclosing object. Where the shift runs
// off the end that surfaces as a read error; where it does not, the record decodes
// to different values and nothing reports it.
func (d Digest) WriteTo(w io.Writer) (n int64, err error) {
	var v = make([]byte, DigestSize)

	switch len(d) {
	case 0:
	case DigestSize:
		copy(v, d)
	default:
		return 0, ErrInvalidDigestLength
	}

	n2, err := w.Write(v)
	return int64(n2), err
}

func (d *Digest) ReadFrom(r io.Reader) (n int64, err error) {
	var v = make([]byte, DigestSize)

	n2, err := io.ReadFull(r, v)
	n = int64(n2)
	if err != nil {
		return
	}

	// all null is the zero value's wire form, not an onion service: a v3 address
	// carries a checksum over its key, and no key checksums to zero.
	if bytes.Equal(v, zeroDigest[:]) {
		*d = nil
		return
	}

	*d = v
	return
}

// text support

// MarshalText renders the digest as String does, so the zero value emits
// unknown -- the one text form UnmarshalText accepts for it. Encoding the bytes
// unconditionally emits a bare .onion, which UnmarshalText then rejects on
// digest length.
func (d Digest) MarshalText() (text []byte, err error) {
	return []byte(d.String()), nil
}

func (d *Digest) UnmarshalText(text []byte) error {
	if string(text) == zeroDigestText {
		*d = nil
		return nil
	}

	var s = strings.ToUpper(string(text))
	s, _ = strings.CutSuffix(s, ".ONION")
	b, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	if len(b) != DigestSize {
		return ErrInvalidDigestLength
	}
	*d = b
	return nil
}

// json support

func (d Digest) MarshalJSON() ([]byte, error) {
	txt, err := d.MarshalText()
	if err != nil {
		return nil, err
	}

	return json.Marshal(string(txt))
}

func (d *Digest) UnmarshalJSON(bytes []byte) (err error) {
	var s string
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return
	}

	return d.UnmarshalText([]byte(s))
}

// other

// String renders the digest as its .onion hostname, and the zero value -- which
// names no onion service -- as unknown, the form mod.tor.endpoint already uses
// for the same absence.
func (d Digest) String() string {
	if len(d) == 0 {
		return zeroDigestText
	}

	return strings.ToLower(base32.StdEncoding.EncodeToString(d)) + ".onion"
}

func init() {
	astral.MustAdd(&Digest{})
}
