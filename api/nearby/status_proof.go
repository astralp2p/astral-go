package nearby

import (
	"crypto/sha256"
	"errors"
	"io"
	"reflect"
	"slices"

	"github.com/astralp2p/astral-go/astral"
	"golang.org/x/crypto/chacha20"
)

// The StatusProof hash labels, as ASCII bytes with no terminator.
const (
	proofHashLabel = "astral/nearby/status/v1"
	proofKeyLabel  = "astral/nearby/proof-key/v1"
)

var errNilAttachment = errors.New("nearby: nil attachment")

// StatusProof binds a status to its sender's node key and its moment. It is the last
// attachment of a status. Only the nearby module appends it, never a status composer.
//
// Nonce is fresh per status and Timestamp is the sender's clock. Sig is a DER ECDSA
// secp256k1 signature (scheme asn1, 8 to 72 bytes) by the node key over ProofHash of
// the other attachments, Nonce and Timestamp. A Stealth status carries Sig wrapped with
// WrapProofSig.
type StatusProof struct {
	Nonce     astral.Nonce
	Timestamp astral.Time
	Sig       astral.Bytes8
}

var _ astral.Object = &StatusProof{}

func (StatusProof) ObjectType() string { return "mod.nearby.status_proof" }

func (p StatusProof) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&p).WriteTo(w)
}

func (p *StatusProof) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(p).ReadFrom(r)
}

// json

func (p StatusProof) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&p).MarshalJSON()
}

func (p *StatusProof) UnmarshalJSON(data []byte) error {
	return astral.Objectify(p).UnmarshalJSON(data)
}

// ProofHash returns the hash a StatusProof signs:
//
//	SHA-256("astral/nearby/status/v1" || B || nonce || ts)
//
// B is the bytes Bundle.WriteTo writes for a Bundle holding others in their order.
// nonce and ts are their 8-byte big-endian wire forms. It fails when others holds a nil
// object (an interface nil or a nil pointer) or two objects with one Object ID.
func ProofHash(others []astral.Object, nonce astral.Nonce, ts astral.Time) ([]byte, error) {
	if slices.ContainsFunc(others, isNilObject) {
		return nil, errNilAttachment
	}

	bundle := astral.NewBundle()
	if err := bundle.Append(others...); err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(proofHashLabel))
	if _, err := bundle.WriteTo(h); err != nil {
		return nil, err
	}
	h.Write(nonceBytes(nonce))
	if _, err := ts.WriteTo(h); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// isNilObject reports whether o is an interface nil or a nil pointer.
//
// why: Bundle.Append calls ObjectType on each object, and a nil pointer to a type that
// declares ObjectType on its value receiver panics there.
func isNilObject(o astral.Object) bool {
	if o == nil {
		return true
	}
	v := reflect.ValueOf(o)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// WrapProofSig returns sig XOR the ChaCha20 keystream of len(sig) bytes under key K,
// a 12-byte all-zero nonce and counter 0, where
//
//	K = SHA-256("astral/nearby/proof-key/v1" || H(N) || nonce)
//
// N is node's compressed key, H is SHA-256, and nonce is the 8-byte big-endian form of
// the proof's Nonce. The same call unwraps. It returns nil when node is nil or zero.
//
// why: a plain ECDSA signature over a publicly computable hash lets anyone recover the
// signer's public key. Wrapped, only a node that already knows N reads it.
func WrapProofSig(node *astral.Identity, nonce astral.Nonce, sig []byte) []byte {
	n := identityKey(node)
	if n == nil {
		return nil
	}

	hn := sha256.Sum256(n)
	key := sha256Of([]byte(proofKeyLabel), hn[:], nonceBytes(nonce))

	var zeroNonce [chacha20.NonceSize]byte
	stream, err := chacha20.NewUnauthenticatedCipher(key, zeroNonce[:])
	if err != nil {
		// unreachable: the key is 32 bytes and the nonce 12
		return nil
	}

	out := make([]byte, len(sig))
	stream.XORKeyStream(out, sig)
	return out
}

// SplitProof splits a status's attachments into the others and the proof. ok is true
// only when the last object is a non-nil *StatusProof and no other object is a
// *StatusProof. others shares objs' elements but not its spare capacity.
func SplitProof(objs []astral.Object) (others []astral.Object, proof *StatusProof, ok bool) {
	if len(objs) == 0 {
		return nil, nil, false
	}

	proof, ok = objs[len(objs)-1].(*StatusProof)
	if !ok || proof == nil {
		return nil, nil, false
	}

	others = slices.Clip(objs[:len(objs)-1])
	for _, o := range others {
		if _, isProof := o.(*StatusProof); isProof {
			return nil, nil, false
		}
	}
	return others, proof, true
}

func init() {
	_ = astral.Add(&StatusProof{})
}
