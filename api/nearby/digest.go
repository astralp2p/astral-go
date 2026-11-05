package nearby

import (
	"crypto/sha256"
	"crypto/subtle"

	"github.com/astralp2p/astral-go/astral"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// identityKeyLen is the length of a compressed secp256k1 public key.
const identityKeyLen = secp256k1.PubKeyBytesLenCompressed

// identityKey returns the 33-byte compressed public key of id, or nil when id is nil
// or zero.
func identityKey(id *astral.Identity) []byte {
	if id.IsZero() {
		return nil
	}
	return id.PublicKey().SerializeCompressed()
}

// parseIdentityKey returns the identity whose compressed public key is key. ok is false
// when key is not a valid compressed point.
func parseIdentityKey(key []byte) (id *astral.Identity, ok bool) {
	if len(key) != identityKeyLen {
		return nil, false
	}
	pub, err := secp256k1.ParsePubKey(key)
	if err != nil {
		return nil, false
	}
	return astral.IdentityFromPubKey(pub), true
}

// nonceBytes returns the 8-byte big-endian wire form of nonce.
func nonceBytes(nonce astral.Nonce) []byte {
	return astral.ByteOrder.AppendUint64(nil, uint64(nonce))
}

// sha256Of returns SHA-256 over the concatenation of parts.
func sha256Of(parts ...[]byte) []byte {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

// xorBytes returns a XOR b. It reads len(a) bytes of b; a shorter b leaves the rest
// of the result zero.
func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	subtle.XORBytes(out, a, b)
	return out
}
