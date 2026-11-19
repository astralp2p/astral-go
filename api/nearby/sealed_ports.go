package nearby

import (
	"bytes"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// portsKeyLabel is the SealedPorts key label, as ASCII bytes with no terminator.
const portsKeyLabel = "astral/nearby/ports-key/v1"

// sealedPortsLen is the length of SealedPorts.Data: the payload of a ListenPorts.
const sealedPortsLen = 4

// SealedPorts is a Stealth status's ListenPorts, readable only by a node that knows the
// sender's node N. Data is the 4-byte ListenPorts payload XOR the ChaCha20 keystream
// under key K, a 12-byte all-zero nonce and counter 0, where
//
//	K = SHA-256("astral/nearby/ports-key/v1" || H(N) || nonce)
//
// H is SHA-256 and nonce is the 8-byte big-endian Nonce of the status's StealthHint.
type SealedPorts struct {
	Data astral.Bytes8
}

var _ astral.Object = &SealedPorts{}

func (SealedPorts) ObjectType() string { return "mod.nearby.sealed_ports" }

func (s SealedPorts) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&s).WriteTo(w)
}

func (s *SealedPorts) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(s).ReadFrom(r)
}

// json

func (s SealedPorts) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&s).MarshalJSON()
}

func (s *SealedPorts) UnmarshalJSON(data []byte) error {
	return astral.Objectify(s).UnmarshalJSON(data)
}

// SealPorts seals p for the nodes that know node, under the nonce of the status's
// StealthHint. It returns astral.ErrZeroIdentity when node is nil or zero.
func SealPorts(node *astral.Identity, nonce astral.Nonce, p *ListenPorts) (*SealedPorts, error) {
	var plain bytes.Buffer
	if _, err := p.WriteTo(&plain); err != nil {
		return nil, err
	}

	data := nodeKeyXOR(portsKeyLabel, node, nonce, plain.Bytes())
	if data == nil {
		return nil, astral.ErrZeroIdentity
	}
	return &SealedPorts{Data: data}, nil
}

// Open returns the ports sealed for node under nonce. ok is false for a nil s, a Data
// that is not 4 bytes, and a nil or zero node.
//
// note: a wrong node or nonce yields wrong ports with ok true; a receiver opens only
// after the status proof verified under node.
func (s *SealedPorts) Open(node *astral.Identity, nonce astral.Nonce) (p *ListenPorts, ok bool) {
	if s == nil || len(s.Data) != sealedPortsLen {
		return nil, false
	}

	plain := nodeKeyXOR(portsKeyLabel, node, nonce, s.Data)
	if plain == nil {
		return nil, false
	}

	p = &ListenPorts{}
	if _, err := p.ReadFrom(bytes.NewReader(plain)); err != nil {
		return nil, false
	}
	return p, true
}

func init() {
	_ = astral.Add(&SealedPorts{})
}
