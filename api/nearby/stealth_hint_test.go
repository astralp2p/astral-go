package nearby

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// refStealthHint computes the hint fields from the definition, independently of the
// package code.
func refStealthHint(n33, u33 []byte, nonce uint64) (commitment, maskedNode, maskedUser []byte) {
	be := make([]byte, 8)
	binary.BigEndian.PutUint64(be, nonce)
	hu := sha256.Sum256(u33)
	hn := sha256.Sum256(n33)

	var in []byte
	in = append(append(append(in, "astral/nearby/stealth/commit/v1"...), hu[:]...), be...)
	c := sha256.Sum256(in)

	in = nil
	in = append(append(append(in, "astral/nearby/stealth/mask-node/v1"...), hu[:]...), be...)
	mn := sha512.Sum512(in)

	in = nil
	in = append(append(append(in, "astral/nearby/stealth/mask-user/v1"...), hn[:]...), be...)
	mu := sha512.Sum512(in)

	maskedNode = make([]byte, 33)
	maskedUser = make([]byte, 33)
	for i := range 33 {
		maskedNode[i] = n33[i] ^ mn[i]
		maskedUser[i] = u33[i] ^ mu[i]
	}
	return c[:], maskedNode, maskedUser
}

type hintCase struct {
	name       string
	node, user *astral.Identity
	nonce      astral.Nonce
}

func hintCases() []hintCase {
	return []hintCase{
		{"fixed keys", testIdentity(1), testIdentity(2), 0x0123456789abcdef},
		{"zero nonce", testIdentity(3), testIdentity(4), 0},
		{"max nonce", testIdentity(5), testIdentity(6), ^astral.Nonce(0)},
		{"random", astral.GenerateIdentity(), astral.GenerateIdentity(), astral.NewNonce()},
		{"node is user", testIdentity(7), testIdentity(7), 42},
	}
}

func mustHint(t *testing.T, node, user *astral.Identity, nonce astral.Nonce) *StealthHint {
	t.Helper()
	h, err := NewStealthHint(node, user, nonce)
	if err != nil {
		t.Fatalf("NewStealthHint: %v", err)
	}
	return h
}

func TestStealthHintMatchesTheDefinition(t *testing.T) {
	for _, tc := range hintCases() {
		t.Run(tc.name, func(t *testing.T) {
			h := mustHint(t, tc.node, tc.user, tc.nonce)

			c, mn, mu := refStealthHint(
				tc.node.PublicKey().SerializeCompressed(),
				tc.user.PublicKey().SerializeCompressed(),
				uint64(tc.nonce),
			)
			if h.Nonce != tc.nonce {
				t.Errorf("Nonce = %v; want %v", h.Nonce, tc.nonce)
			}
			if !bytes.Equal(h.Commitment, c) {
				t.Errorf("Commitment = %x; want %x", []byte(h.Commitment), c)
			}
			if !bytes.Equal(h.MaskedNode, mn) {
				t.Errorf("MaskedNode = %x; want %x", []byte(h.MaskedNode), mn)
			}
			if !bytes.Equal(h.MaskedUser, mu) {
				t.Errorf("MaskedUser = %x; want %x", []byte(h.MaskedUser), mu)
			}
		})
	}
}

// Two hints of one node and user under different nonces share no field value.
func TestStealthHintsDifferPerNonce(t *testing.T) {
	node, user := testIdentity(1), testIdentity(2)
	a, b := mustHint(t, node, user, 1), mustHint(t, node, user, 2)

	if bytes.Equal(a.Commitment, b.Commitment) || bytes.Equal(a.MaskedNode, b.MaskedNode) ||
		bytes.Equal(a.MaskedUser, b.MaskedUser) {
		t.Errorf("hints under nonces 1 and 2 share a field:\n%+v\n%+v", a, b)
	}
}

func TestNewStealthHintRefusesAbsentIdentities(t *testing.T) {
	id := testIdentity(1)
	tests := []struct {
		name       string
		node, user *astral.Identity
	}{
		{"nil node", nil, id},
		{"nil user", id, nil},
		{"zero node", &astral.Identity{}, id},
		{"zero user", id, &astral.Identity{}},
		{"both nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustNotPanic(t, "NewStealthHint", func() {
				h, err := NewStealthHint(tt.node, tt.user, 1)
				if !errors.Is(err, astral.ErrZeroIdentity) || h != nil {
					t.Errorf("NewStealthHint = %v, %v; want nil, ErrZeroIdentity", h, err)
				}
			})
		})
	}
}

func TestStealthHintOpens(t *testing.T) {
	for _, tc := range hintCases() {
		t.Run(tc.name, func(t *testing.T) {
			h := mustHint(t, tc.node, tc.user, tc.nonce)

			node, ok := h.Open(tc.user)
			if !ok || !node.IsEqual(tc.node) {
				t.Errorf("Open(user) = %v, %v; want %v, true", node, ok, tc.node)
			}

			user, ok := h.OpenWithNode(tc.node)
			if !ok || !user.IsEqual(tc.user) {
				t.Errorf("OpenWithNode(node) = %v, %v; want %v, true", user, ok, tc.user)
			}
		})
	}
}

// assertClosed checks that h opens for none of the given identities, and that no call
// panics.
func assertClosed(t *testing.T, h *StealthHint, ids ...*astral.Identity) {
	t.Helper()
	for _, id := range ids {
		mustNotPanic(t, "Open", func() {
			if got, ok := h.Open(id); ok || got != nil {
				t.Errorf("Open(%v) = %v, %v; want nil, false", id, got, ok)
			}
		})
		mustNotPanic(t, "OpenWithNode", func() {
			if got, ok := h.OpenWithNode(id); ok || got != nil {
				t.Errorf("OpenWithNode(%v) = %v, %v; want nil, false", id, got, ok)
			}
		})
	}
}

func TestStealthHintRefusesOtherIdentities(t *testing.T) {
	node, user, other := testIdentity(1), testIdentity(2), testIdentity(3)
	h := mustHint(t, node, user, 9)

	mustNotPanic(t, "Open", func() {
		if got, ok := h.Open(other); ok {
			t.Errorf("Open(other user) = %v, true; want false", got)
		}
		if got, ok := h.Open(node); ok {
			t.Errorf("Open(node) = %v, true; want false", got)
		}
	})
	mustNotPanic(t, "OpenWithNode", func() {
		if got, ok := h.OpenWithNode(other); ok {
			t.Errorf("OpenWithNode(other node) = %v, true; want false", got)
		}
		if got, ok := h.OpenWithNode(user); ok {
			t.Errorf("OpenWithNode(user) = %v, true; want false", got)
		}
	})
}

func TestStealthHintRefusesAbsentIdentities(t *testing.T) {
	h := mustHint(t, testIdentity(1), testIdentity(2), 9)
	assertClosed(t, h, nil, &astral.Identity{}, astral.Anyone)
}

func cloneHint(h *StealthHint) *StealthHint {
	return &StealthHint{
		Nonce:      h.Nonce,
		Commitment: bytes.Clone(h.Commitment),
		MaskedNode: bytes.Clone(h.MaskedNode),
		MaskedUser: bytes.Clone(h.MaskedUser),
	}
}

// Changing any bit pattern of any field closes the hint for both openers.
func TestStealthHintRefusesTamperedFields(t *testing.T) {
	node, user := testIdentity(1), testIdentity(2)
	h := mustHint(t, node, user, 0x0123456789abcdef)

	fields := map[string]func(*StealthHint) astral.Bytes8{
		"Commitment": func(s *StealthHint) astral.Bytes8 { return s.Commitment },
		"MaskedNode": func(s *StealthHint) astral.Bytes8 { return s.MaskedNode },
		"MaskedUser": func(s *StealthHint) astral.Bytes8 { return s.MaskedUser },
	}
	for name, field := range fields {
		for i := range field(h) {
			for _, flip := range []byte{0x01, 0x80} {
				t.Run(fmt.Sprintf("%s[%d]^%#x", name, i, flip), func(t *testing.T) {
					tampered := cloneHint(h)
					field(tampered)[i] ^= flip
					assertClosed(t, tampered, node, user)
				})
			}
		}
	}

	for _, nonce := range []astral.Nonce{h.Nonce + 1, h.Nonce ^ 1<<63, 0} {
		t.Run(fmt.Sprintf("Nonce=%x", uint64(nonce)), func(t *testing.T) {
			tampered := cloneHint(h)
			tampered.Nonce = nonce
			assertClosed(t, tampered, node, user)
		})
	}
}

func TestStealthHintRefusesWrongLengths(t *testing.T) {
	node, user := testIdentity(1), testIdentity(2)
	h := mustHint(t, node, user, 7)

	resize := func(b astral.Bytes8, n int) astral.Bytes8 {
		out := make(astral.Bytes8, n)
		copy(out, b)
		return out
	}

	tests := []struct {
		name   string
		mutate func(*StealthHint)
	}{
		{"nil Commitment", func(s *StealthHint) { s.Commitment = nil }},
		{"short Commitment", func(s *StealthHint) { s.Commitment = s.Commitment[:31] }},
		{"long Commitment", func(s *StealthHint) { s.Commitment = resize(s.Commitment, 33) }},
		{"nil MaskedNode", func(s *StealthHint) { s.MaskedNode = nil }},
		{"short MaskedNode", func(s *StealthHint) { s.MaskedNode = s.MaskedNode[:32] }},
		{"long MaskedNode", func(s *StealthHint) { s.MaskedNode = resize(s.MaskedNode, 34) }},
		{"nil MaskedUser", func(s *StealthHint) { s.MaskedUser = nil }},
		{"short MaskedUser", func(s *StealthHint) { s.MaskedUser = s.MaskedUser[:32] }},
		{"long MaskedUser", func(s *StealthHint) { s.MaskedUser = resize(s.MaskedUser, 34) }},
		{"empty fields", func(s *StealthHint) { *s = StealthHint{Nonce: s.Nonce} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := cloneHint(h)
			tt.mutate(tampered)
			assertClosed(t, tampered, node, user)
		})
	}
}

func TestStealthHintNilAndZeroHintsStayClosed(t *testing.T) {
	node, user := testIdentity(1), testIdentity(2)
	assertClosed(t, nil, node, user, nil)
	assertClosed(t, &StealthHint{}, node, user, nil)
}

// Random fields of the right lengths, and random bytes decoded off the wire, never
// open and never panic.
func TestStealthHintGarbageStaysClosed(t *testing.T) {
	node, user := testIdentity(1), testIdentity(2)

	random := func(n int) []byte {
		b := make([]byte, n)
		_, _ = rand.Read(b)
		return b
	}

	for range 200 {
		h := &StealthHint{
			Nonce:      astral.NewNonce(),
			Commitment: random(32),
			MaskedNode: random(33),
			MaskedUser: random(33),
		}
		assertClosed(t, h, node, user)

		var decoded StealthHint
		mustNotPanic(t, "ReadFrom", func() {
			if _, err := decoded.ReadFrom(bytes.NewReader(random(109))); err == nil {
				assertClosed(t, &decoded, node, user)
			}
		})
	}
}

func TestStealthHintBinaryRoundTrip(t *testing.T) {
	for _, tc := range hintCases() {
		t.Run(tc.name, func(t *testing.T) {
			orig := mustHint(t, tc.node, tc.user, tc.nonce)
			data, err := astral.EncodeBytes(orig)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			got, err := astral.DecodeAs[*StealthHint](data)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			assertHintEqual(t, got, orig)
			if node, ok := got.Open(tc.user); !ok || !node.IsEqual(tc.node) {
				t.Errorf("decoded hint Open = %v, %v; want %v, true", node, ok, tc.node)
			}
		})
	}
}

func TestStealthHintJSONRoundTrip(t *testing.T) {
	for _, tc := range hintCases() {
		t.Run(tc.name, func(t *testing.T) {
			orig := mustHint(t, tc.node, tc.user, tc.nonce)
			data, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got StealthHint
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", data, err)
			}
			assertHintEqual(t, &got, orig)
		})
	}
}

func assertHintEqual(t *testing.T, got, want *StealthHint) {
	t.Helper()
	if got.Nonce != want.Nonce || !bytes.Equal(got.Commitment, want.Commitment) ||
		!bytes.Equal(got.MaskedNode, want.MaskedNode) || !bytes.Equal(got.MaskedUser, want.MaskedUser) {
		t.Errorf("round trip = %+v; want %+v", got, want)
	}
}

// The fields sit on the wire in the order Nonce, Commitment, MaskedNode, MaskedUser:
// 8 + (1+32) + (1+33) + (1+33) = 109 bytes.
func TestStealthHintWireLayout(t *testing.T) {
	h := mustHint(t, testIdentity(1), testIdentity(2), 0x0102030405060708)

	var buf bytes.Buffer
	if _, err := h.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	want := bytes.Join([][]byte{
		{1, 2, 3, 4, 5, 6, 7, 8},
		{32}, h.Commitment,
		{33}, h.MaskedNode,
		{33}, h.MaskedUser,
	}, nil)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("wire = %x; want %x", buf.Bytes(), want)
	}
	if buf.Len() != 109 {
		t.Errorf("wire length = %d; want 109", buf.Len())
	}
}
