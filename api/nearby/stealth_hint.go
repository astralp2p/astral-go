package nearby

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// The StealthHint hash labels, as ASCII bytes with no terminator.
//
// why: separate labels. With one label MaskedNode XOR Commitment would reveal the node.
const (
	stealthCommitLabel   = "astral/nearby/stealth/commit/v1"
	stealthMaskNodeLabel = "astral/nearby/stealth/mask-node/v1"
	stealthMaskUserLabel = "astral/nearby/stealth/mask-user/v1"
)

// StealthHint is a Stealth status's identity. A node that knows the user U recovers the
// node N from it, a node that knows N recovers U, and others learn neither.
//
//	Commitment = SHA-256(commit label || H(U) || nonce)                 32 bytes
//	MaskedNode = N XOR SHA-512(mask-node label || H(U) || nonce)[0:33]  33 bytes
//	MaskedUser = U XOR SHA-512(mask-user label || H(N) || nonce)[0:33]  33 bytes
//
// N and U are 33-byte compressed public keys, H(X) is SHA-256(X), and nonce is the
// 8-byte big-endian Nonce. A hint takes a fresh Nonce.
//
// why: SHA-512 truncated to 33 bytes. SHA-256 gives 32, and an unmasked 33rd byte
// would leak a key bit.
type StealthHint struct {
	Nonce      astral.Nonce
	Commitment astral.Bytes8
	MaskedNode astral.Bytes8
	MaskedUser astral.Bytes8
}

var _ astral.Object = &StealthHint{}

func (StealthHint) ObjectType() string { return "mod.nearby.stealth_hint" }

func (s StealthHint) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&s).WriteTo(w)
}

func (s *StealthHint) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(s).ReadFrom(r)
}

// json

func (s StealthHint) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&s).MarshalJSON()
}

func (s *StealthHint) UnmarshalJSON(data []byte) error {
	return astral.Objectify(s).UnmarshalJSON(data)
}

// NewStealthHint returns the hint of node owned by user under nonce. It returns
// astral.ErrZeroIdentity when node or user is nil or zero.
func NewStealthHint(node, user *astral.Identity, nonce astral.Nonce) (*StealthHint, error) {
	n, u := identityKey(node), identityKey(user)
	if n == nil || u == nil {
		return nil, astral.ErrZeroIdentity
	}

	hn, hu := sha256.Sum256(n), sha256.Sum256(u)
	return &StealthHint{
		Nonce:      nonce,
		Commitment: stealthCommitment(hu[:], nonce),
		MaskedNode: xorBytes(n, stealthMask(stealthMaskNodeLabel, hu[:], nonce)),
		MaskedUser: xorBytes(u, stealthMask(stealthMaskUserLabel, hn[:], nonce)),
	}, nil
}

// Open returns the node of a hint made for user. ok is false unless the commitment
// names user, MaskedNode unmasks to a valid key, and MaskedUser unmasks to user. A nil
// or malformed hint and a nil or zero user return false.
func (s *StealthHint) Open(user *astral.Identity) (node *astral.Identity, ok bool) {
	u := identityKey(user)
	if !s.wellFormed() || u == nil {
		return nil, false
	}

	hu := sha256.Sum256(u)
	if subtle.ConstantTimeCompare(s.Commitment, stealthCommitment(hu[:], s.Nonce)) != 1 {
		return nil, false
	}

	n := xorBytes(s.MaskedNode, stealthMask(stealthMaskNodeLabel, hu[:], s.Nonce))
	node, ok = parseIdentityKey(n)
	if !ok {
		return nil, false
	}

	hn := sha256.Sum256(n)
	if !bytes.Equal(xorBytes(s.MaskedUser, stealthMask(stealthMaskUserLabel, hn[:], s.Nonce)), u) {
		return nil, false
	}
	return node, true
}

// OpenWithNode returns the user of a hint made for node. ok is false unless MaskedUser
// unmasks to a valid key, the commitment names that user, and MaskedNode unmasks to
// node. A nil or malformed hint and a nil or zero node return false.
func (s *StealthHint) OpenWithNode(node *astral.Identity) (user *astral.Identity, ok bool) {
	n := identityKey(node)
	if !s.wellFormed() || n == nil {
		return nil, false
	}

	hn := sha256.Sum256(n)
	u := xorBytes(s.MaskedUser, stealthMask(stealthMaskUserLabel, hn[:], s.Nonce))
	user, ok = parseIdentityKey(u)
	if !ok {
		return nil, false
	}

	hu := sha256.Sum256(u)
	if subtle.ConstantTimeCompare(s.Commitment, stealthCommitment(hu[:], s.Nonce)) != 1 {
		return nil, false
	}
	if !bytes.Equal(xorBytes(s.MaskedNode, stealthMask(stealthMaskNodeLabel, hu[:], s.Nonce)), n) {
		return nil, false
	}
	return user, true
}

// wellFormed reports whether s is non-nil and its fields have the lengths 32, 33, 33.
func (s *StealthHint) wellFormed() bool {
	return s != nil &&
		len(s.Commitment) == sha256.Size &&
		len(s.MaskedNode) == identityKeyLen &&
		len(s.MaskedUser) == identityKeyLen
}

// stealthCommitment returns SHA-256(commit label || hu || nonce).
func stealthCommitment(hu []byte, nonce astral.Nonce) []byte {
	return sha256Of([]byte(stealthCommitLabel), hu, nonceBytes(nonce))
}

// stealthMask returns the first 33 bytes of SHA-512(label || h || nonce).
func stealthMask(label string, h []byte, nonce astral.Nonce) []byte {
	sum := sha512.New()
	sum.Write([]byte(label))
	sum.Write(h)
	sum.Write(nonceBytes(nonce))
	return sum.Sum(nil)[:identityKeyLen]
}

func init() {
	_ = astral.Add(&StealthHint{})
}
