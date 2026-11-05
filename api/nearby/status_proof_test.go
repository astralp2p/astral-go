package nearby

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	decredecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/chacha20"
)

// refMember is one bundle member, encoded by hand: its type name and payload bytes.
type refMember struct {
	typ     string
	payload []byte
}

// refBundle encodes members as Bundle.WriteTo defines it: a uint32 count, then each
// member as bytes32(string8 type || payload).
func refBundle(members []refMember) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(members)))
	for _, m := range members {
		block := append([]byte{byte(len(m.typ))}, m.typ...)
		block = append(block, m.payload...)
		out = binary.BigEndian.AppendUint32(out, uint32(len(block)))
		out = append(out, block...)
	}
	return out
}

// refProofHash computes H from the definition, independently of the package code.
func refProofHash(members []refMember, nonce uint64, ts time.Time) []byte {
	in := []byte("astral/nearby/status/v1")
	in = append(in, refBundle(members)...)
	in = binary.BigEndian.AppendUint64(in, nonce)
	in = binary.BigEndian.AppendUint64(in, uint64(ts.UnixNano()))
	h := sha256.Sum256(in)
	return h[:]
}

// refWrap computes the wrapped signature from the definition, independently of the
// package code.
func refWrap(t *testing.T, n33 []byte, nonce uint64, sig []byte) []byte {
	t.Helper()
	hn := sha256.Sum256(n33)
	in := append([]byte("astral/nearby/proof-key/v1"), hn[:]...)
	in = binary.BigEndian.AppendUint64(in, nonce)
	key := sha256.Sum256(in)

	c, err := chacha20.NewUnauthenticatedCipher(key[:], make([]byte, 12))
	if err != nil {
		t.Fatalf("chacha20: %v", err)
	}
	keystream := make([]byte, len(sig))
	c.XORKeyStream(keystream, keystream)

	out := make([]byte, len(sig))
	for i := range sig {
		out[i] = sig[i] ^ keystream[i]
	}
	return out
}

func profileMember(p *PublicProfile) refMember {
	var payload []byte
	for _, id := range []*astral.Identity{p.NodeID, p.UserID} {
		if id == nil {
			payload = append(payload, 0)
			continue
		}
		payload = append(append(payload, 1), id.PublicKey().SerializeCompressed()...)
	}
	payload = append(append(payload, byte(len(p.NodeAlias))), p.NodeAlias...)
	return refMember{"mod.nearby.public_profile", payload}
}

func hintMember(h *StealthHint) refMember {
	payload := binary.BigEndian.AppendUint64(nil, uint64(h.Nonce))
	for _, f := range [][]byte{h.Commitment, h.MaskedNode, h.MaskedUser} {
		payload = append(append(payload, byte(len(f))), f...)
	}
	return refMember{"mod.nearby.stealth_hint", payload}
}

func flagMember(f string) refMember {
	return refMember{"mod.nearby.flag", append([]byte{byte(len(f))}, f...)}
}

type proofHashCase struct {
	name    string
	others  []astral.Object
	members []refMember
}

func proofHashCases(t *testing.T) []proofHashCase {
	node, user := testIdentity(1), testIdentity(2)
	owned := &PublicProfile{NodeID: node, UserID: user, NodeAlias: "alice-laptop"}
	unowned := &PublicProfile{NodeID: node, NodeAlias: "new-node"}
	hint := mustHint(t, node, user, 77)

	return []proofHashCase{
		{"no attachments", nil, nil},
		{"visible owned", []astral.Object{owned}, []refMember{profileMember(owned)}},
		{"visible unowned", []astral.Object{NewFlag("adoptable"), unowned},
			[]refMember{flagMember("adoptable"), profileMember(unowned)}},
		{"stealth", []astral.Object{hint}, []refMember{hintMember(hint)}},
	}
}

var proofTime = astral.Time(time.Date(2026, 11, 5, 12, 0, 0, 123456789, time.UTC))

func TestProofHashMatchesTheDefinition(t *testing.T) {
	for _, tc := range proofHashCases(t) {
		for _, nonce := range []astral.Nonce{0, 0x0123456789abcdef, ^astral.Nonce(0)} {
			t.Run(fmt.Sprintf("%s/nonce=%x", tc.name, uint64(nonce)), func(t *testing.T) {
				got, err := ProofHash(tc.others, nonce, proofTime)
				if err != nil {
					t.Fatalf("ProofHash: %v", err)
				}
				want := refProofHash(tc.members, uint64(nonce), time.Time(proofTime))
				if !bytes.Equal(got, want) {
					t.Errorf("ProofHash = %x; want %x", got, want)
				}
			})
		}
	}
}

// Every input of the hash changes it.
func TestProofHashCoversItsInputs(t *testing.T) {
	a := &PublicProfile{NodeID: testIdentity(1), NodeAlias: "a"}
	b := NewFlag("adoptable")
	base, err := ProofHash([]astral.Object{b, a}, 1, proofTime)
	if err != nil {
		t.Fatal(err)
	}

	later := astral.Time(time.Time(proofTime).Add(time.Nanosecond))
	renamed := &PublicProfile{NodeID: testIdentity(1), NodeAlias: "b"}
	variants := map[string]func() ([]byte, error){
		"nonce":     func() ([]byte, error) { return ProofHash([]astral.Object{b, a}, 2, proofTime) },
		"timestamp": func() ([]byte, error) { return ProofHash([]astral.Object{b, a}, 1, later) },
		"order":     func() ([]byte, error) { return ProofHash([]astral.Object{a, b}, 1, proofTime) },
		"content":   func() ([]byte, error) { return ProofHash([]astral.Object{b, renamed}, 1, proofTime) },
		"dropped":   func() ([]byte, error) { return ProofHash([]astral.Object{a}, 1, proofTime) },
	}
	for name, hash := range variants {
		got, err := hash()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if bytes.Equal(got, base) {
			t.Errorf("changing the %s leaves ProofHash unchanged", name)
		}
	}
}

func TestProofHashRefusesUnbundleableAttachments(t *testing.T) {
	flag := NewFlag("adoptable")
	tests := map[string][]astral.Object{
		"nil object":     {flag, nil},
		"duplicate flag": {flag, NewFlag("adoptable")},
	}
	for name, others := range tests {
		t.Run(name, func(t *testing.T) {
			mustNotPanic(t, "ProofHash", func() {
				if h, err := ProofHash(others, 1, proofTime); err == nil {
					t.Errorf("ProofHash = %x, nil; want an error", h)
				}
			})
		})
	}
}

func TestWrapProofSigMatchesTheDefinition(t *testing.T) {
	node := testIdentity(1)
	for _, n := range []int{0, 1, 8, 64, 70, 71, 72, 255} {
		for _, nonce := range []astral.Nonce{0, 0x0123456789abcdef} {
			t.Run(fmt.Sprintf("len=%d/nonce=%x", n, uint64(nonce)), func(t *testing.T) {
				sig := make([]byte, n)
				_, _ = rand.Read(sig)

				got := WrapProofSig(node, nonce, sig)
				want := refWrap(t, node.PublicKey().SerializeCompressed(), uint64(nonce), sig)
				if !bytes.Equal(got, want) {
					t.Errorf("WrapProofSig = %x; want %x", got, want)
				}
			})
		}
	}
}

// The wrap is its own inverse, changes the bytes, keeps the length, leaves its input
// alone, and depends on both the node and the nonce.
func TestWrapProofSigIsAnInvolution(t *testing.T) {
	node, other := testIdentity(1), testIdentity(2)
	for _, n := range []int{8, 70, 71, 72} {
		t.Run(fmt.Sprintf("len=%d", n), func(t *testing.T) {
			sig := make([]byte, n)
			_, _ = rand.Read(sig)
			orig := bytes.Clone(sig)

			wrapped := WrapProofSig(node, 5, sig)
			if !bytes.Equal(sig, orig) {
				t.Fatal("WrapProofSig modified its input")
			}
			if len(wrapped) != n {
				t.Errorf("len(wrapped) = %d; want %d", len(wrapped), n)
			}
			if bytes.Equal(wrapped, sig) {
				t.Error("wrapped signature equals the plain one")
			}
			if back := WrapProofSig(node, 5, wrapped); !bytes.Equal(back, sig) {
				t.Errorf("wrap of the wrapped = %x; want %x", back, sig)
			}
			if bytes.Equal(WrapProofSig(other, 5, sig), wrapped) {
				t.Error("another node wraps to the same bytes")
			}
			if bytes.Equal(WrapProofSig(node, 6, sig), wrapped) {
				t.Error("another nonce wraps to the same bytes")
			}
		})
	}
}

func TestWrapProofSigRefusesAbsentNodes(t *testing.T) {
	for name, node := range map[string]*astral.Identity{"nil": nil, "zero": {}, "anyone": astral.Anyone} {
		mustNotPanic(t, name, func() {
			if got := WrapProofSig(node, 1, []byte("signature")); got != nil {
				t.Errorf("%s node: WrapProofSig = %x; want nil", name, got)
			}
		})
	}
}

// signProof signs hash with priv the way astrald's node signer does (scheme asn1).
func signProof(t *testing.T, priv *secp256k1.PrivateKey, hash []byte) []byte {
	t.Helper()
	sig, err := ecdsa.SignASN1(rand.Reader, priv.ToECDSA(), hash)
	if err != nil {
		t.Fatalf("SignASN1: %v", err)
	}
	if len(sig) < 8 || len(sig) > 72 {
		t.Fatalf("len(sig) = %d; want 8..72", len(sig))
	}
	return sig
}

// verifies reports whether der verifies over hash under pub, by both the standard
// library and decred.
func verifies(t *testing.T, pub *secp256k1.PublicKey, hash, der []byte) bool {
	t.Helper()
	std := ecdsa.VerifyASN1(pub.ToECDSA(), hash, der)

	dec := false
	if sig, err := decredecdsa.ParseDERSignature(der); err == nil {
		dec = sig.Verify(hash, pub)
	}
	if std != dec {
		t.Errorf("stdlib verify = %v but decred verify = %v", std, dec)
	}
	return std
}

func TestStatusProofSignatureRoundTrip(t *testing.T) {
	priv, node := testKey(1)
	pub := priv.PubKey()

	for _, tc := range proofHashCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			nonce := astral.NewNonce()
			hash, err := ProofHash(tc.others, nonce, proofTime)
			if err != nil {
				t.Fatal(err)
			}
			sig := signProof(t, priv, hash)

			if !verifies(t, pub, hash, sig) {
				t.Fatal("the plain signature does not verify")
			}

			wrapped := WrapProofSig(node, nonce, sig)
			if verifies(t, pub, hash, wrapped) {
				t.Error("the wrapped signature verifies")
			}

			if !verifies(t, pub, hash, WrapProofSig(node, nonce, wrapped)) {
				t.Error("the unwrapped signature does not verify")
			}
			if verifies(t, testIdentity(2).PublicKey(), hash, sig) {
				t.Error("the signature verifies under another key")
			}
		})
	}
}

// recoverFrom tries the four recovery codes on the scalars r and s over hash and
// reports whether one yields want.
func recoverFrom(r, s, hash []byte, want *secp256k1.PublicKey) bool {
	compact := make([]byte, 65)
	copy(compact[1:33], r)
	copy(compact[33:], s)
	for code := byte(0); code < 4; code++ {
		compact[0] = 27 + 4 + code
		pub, _, err := decredecdsa.RecoverCompact(compact, hash)
		if err == nil && pub.IsEqual(want) {
			return true
		}
	}
	return false
}

// recoversSigner reports whether anyone holding sig and hash recovers want: from the
// parsed DER scalars, or, when sig does not parse, from the positions DER gives R and
// S in a 70- to 72-byte signature.
func recoversSigner(sig, hash []byte, want *secp256k1.PublicKey) bool {
	if parsed, err := decredecdsa.ParseDERSignature(sig); err == nil {
		r, s := parsed.R(), parsed.S()
		var rb, sb [32]byte
		r.PutBytesUnchecked(rb[:])
		s.PutBytesUnchecked(sb[:])
		if recoverFrom(rb[:], sb[:], hash, want) {
			return true
		}
	}
	if len(sig) < 68 {
		return false
	}
	s := sig[len(sig)-32:]
	for _, at := range []int{4, 5} {
		if recoverFrom(sig[at:at+32], s, hash, want) {
			return true
		}
	}
	return false
}

// A plain signature over a public hash gives away the signer's key; a wrapped one
// does not.
func TestStatusProofWrapHidesTheSigner(t *testing.T) {
	for seed := byte(1); seed <= 8; seed++ {
		t.Run(fmt.Sprintf("key=%d", seed), func(t *testing.T) {
			priv, node := testKey(seed)
			nonce := astral.NewNonce()
			hash, err := ProofHash([]astral.Object{mustHint(t, node, testIdentity(99), nonce)}, nonce, proofTime)
			if err != nil {
				t.Fatal(err)
			}
			sig := signProof(t, priv, hash)

			if !recoversSigner(sig, hash, priv.PubKey()) {
				t.Fatal("no recovery code yields the signer from the plain signature")
			}
			if recoversSigner(WrapProofSig(node, nonce, sig), hash, priv.PubKey()) {
				t.Error("the wrapped signature yields the signer's key")
			}
		})
	}
}

func TestSplitProof(t *testing.T) {
	proof := &StatusProof{Nonce: 1, Sig: astral.Bytes8{1, 2, 3, 4, 5, 6, 7, 8}}
	second := &StatusProof{Nonce: 2}
	profile := &PublicProfile{NodeID: testIdentity(1)}
	flag := NewFlag("adoptable")

	tests := []struct {
		name       string
		objs       []astral.Object
		wantOK     bool
		wantOthers []astral.Object
	}{
		{"nil", nil, false, nil},
		{"empty", []astral.Object{}, false, nil},
		{"no proof", []astral.Object{flag, profile}, false, nil},
		{"proof not last", []astral.Object{proof, profile}, false, nil},
		{"two proofs at the end", []astral.Object{profile, second, proof}, false, nil},
		{"two proofs apart", []astral.Object{second, profile, proof}, false, nil},
		{"nil proof last", []astral.Object{profile, (*StatusProof)(nil)}, false, nil},
		{"proof only", []astral.Object{proof}, true, []astral.Object{}},
		{"profile and proof", []astral.Object{profile, proof}, true, []astral.Object{profile}},
		{"flag, profile and proof", []astral.Object{flag, profile, proof}, true, []astral.Object{flag, profile}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			others, got, ok := SplitProof(tt.objs)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v; want %v", ok, tt.wantOK)
			}
			if !ok {
				if others != nil || got != nil {
					t.Errorf("SplitProof = %v, %v, false; want nil, nil, false", others, got)
				}
				return
			}
			if got != proof {
				t.Errorf("proof = %p; want %p", got, proof)
			}
			if len(others) != len(tt.wantOthers) {
				t.Fatalf("others = %v; want %v", others, tt.wantOthers)
			}
			for i := range others {
				if others[i] != tt.wantOthers[i] {
					t.Errorf("others[%d] = %v; want %v", i, others[i], tt.wantOthers[i])
				}
			}

			_ = append(others, flag)
			if tt.objs[len(tt.objs)-1] != proof {
				t.Error("appending to others overwrote the proof in objs")
			}
		})
	}
}

func sampleStatusProofs() map[string]*StatusProof {
	der := bytes.Repeat([]byte{0x30}, 72)
	return map[string]*StatusProof{
		"72-byte sig": {Nonce: 0x0123456789abcdef, Timestamp: proofTime, Sig: der},
		"8-byte sig":  {Nonce: 1, Timestamp: proofTime, Sig: der[:8]},
		"no sig":      {Nonce: 2, Timestamp: proofTime},
	}
}

func assertProofEqual(t *testing.T, got, want *StatusProof) {
	t.Helper()
	if got.Nonce != want.Nonce || !time.Time(got.Timestamp).Equal(time.Time(want.Timestamp)) ||
		!bytes.Equal(got.Sig, want.Sig) {
		t.Errorf("round trip = %+v; want %+v", got, want)
	}
}

func TestStatusProofBinaryRoundTrip(t *testing.T) {
	for name, orig := range sampleStatusProofs() {
		t.Run(name, func(t *testing.T) {
			data, err := astral.EncodeBytes(orig)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			got, err := astral.DecodeAs[*StatusProof](data)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			assertProofEqual(t, got, orig)
		})
	}
}

func TestStatusProofJSONRoundTrip(t *testing.T) {
	for name, orig := range sampleStatusProofs() {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got StatusProof
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", data, err)
			}
			assertProofEqual(t, &got, orig)
		})
	}
}

// The fields sit on the wire in the order Nonce, Timestamp, Sig.
func TestStatusProofWireLayout(t *testing.T) {
	p := StatusProof{Nonce: 0x0102030405060708, Timestamp: proofTime, Sig: astral.Bytes8{9, 9}}

	var buf bytes.Buffer
	if _, err := p.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	want := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	want = binary.BigEndian.AppendUint64(want, uint64(time.Time(proofTime).UnixNano()))
	want = append(want, 2, 9, 9)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("wire = %x; want %x", buf.Bytes(), want)
	}
}
