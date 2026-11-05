package nearby

import (
	"bytes"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/api/ether"
	"github.com/astralp2p/astral-go/astral"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// datagramCap is the ether datagram cap on send and receive.
const datagramCap = 1024

// statusDatagram returns the ether datagram carrying a status of attachments: an
// Envelope holding a StatusMessage.
func statusDatagram(t *testing.T, attachments ...astral.Object) []byte {
	t.Helper()
	bundle := astral.NewBundle()
	if err := bundle.Append(attachments...); err != nil {
		t.Fatalf("Append: %v", err)
	}

	env := ether.Envelope{Version: ether.EnvelopeVersion, Object: &StatusMessage{Attachments: bundle}}
	var buf bytes.Buffer
	if _, err := env.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return buf.Bytes()
}

// maxSizeProof returns a StatusProof carrying the longest signature, 72 bytes.
func maxSizeProof() *StatusProof {
	return &StatusProof{Nonce: astral.NewNonce(), Timestamp: astral.Now(), Sig: bytes.Repeat([]byte{0x30}, 72)}
}

// The encoded sizes of the statuses a node sends, and the largest any composition
// reaches, against the datagram cap.
func TestStatusDatagramSizes(t *testing.T) {
	node, user := testIdentity(1), testIdentity(2)

	visible := statusDatagram(t,
		&PublicProfile{NodeID: node, UserID: user, NodeAlias: astral.String8(strings.Repeat("a", 16))},
		maxSizeProof(),
	)
	visibleMax := statusDatagram(t,
		&PublicProfile{NodeID: node, UserID: user, NodeAlias: astral.String8(strings.Repeat("a", 255))},
		maxSizeProof(),
	)
	stealth := statusDatagram(t, mustHint(t, node, user, astral.NewNonce()), maxSizeProof())
	// why: no real status carries both the adoptable flag and a UserID; holding both
	// bounds the visible owned and visible unowned compositions at once.
	worst := statusDatagram(t,
		NewFlag("adoptable"),
		&PublicProfile{NodeID: node, UserID: user, NodeAlias: astral.String8(strings.Repeat("a", 255))},
		maxSizeProof(),
	)

	t.Logf("datagram sizes: visible owned %d B (16-byte alias), %d B (255-byte alias); stealth %d B; "+
		"worst case %d B (flag, UserID, 255-byte alias); cap %d B",
		len(visible), len(visibleMax), len(stealth), len(worst), datagramCap)

	if len(worst) > datagramCap {
		t.Errorf("worst case = %d B; want <= %d B", len(worst), datagramCap)
	}
	for _, sent := range [][]byte{visible, visibleMax, stealth} {
		if len(sent) > len(worst) {
			t.Errorf("a sent status of %d B exceeds the worst case %d B", len(sent), len(worst))
		}
	}
}

// receiveStatus decodes a status datagram as a receiver does and returns its
// attachments split around the proof.
func receiveStatus(t *testing.T, datagram []byte) ([]astral.Object, *StatusProof) {
	t.Helper()
	var env ether.Envelope
	n, err := env.ReadFrom(bytes.NewReader(datagram))
	if err != nil || n != int64(len(datagram)) || env.Version != ether.EnvelopeVersion {
		t.Fatalf("ReadFrom = %d, %v (version %d); want %d, nil (version 1)", n, err, env.Version, len(datagram))
	}
	msg, ok := env.Object.(*StatusMessage)
	if !ok {
		t.Fatalf("Object = %T; want *StatusMessage", env.Object)
	}
	others, proof, ok := SplitProof(msg.Attachments.Objects())
	if !ok {
		t.Fatal("SplitProof: no valid proof")
	}
	return others, proof
}

// signStatus composes a status of others with a signed proof, wrapped when stealth.
func signStatus(t *testing.T, priv *secp256k1.PrivateKey, stealth bool, others ...astral.Object) []astral.Object {
	t.Helper()
	proof := &StatusProof{Nonce: astral.NewNonce(), Timestamp: astral.Now()}
	hash, err := ProofHash(others, proof.Nonce, proof.Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	proof.Sig = signProof(t, priv, hash)
	if stealth {
		proof.Sig = WrapProofSig(astral.IdentityFromPubKey(priv.PubKey()), proof.Nonce, proof.Sig)
	}
	return append(others, proof)
}

// A receiver recomputes the signed hash from the decoded attachments and verifies the
// proof under the identified node key.
func TestStatusDatagramVerifiesAfterTheWire(t *testing.T) {
	priv, node := testKey(1)
	user := testIdentity(2)

	t.Run("visible", func(t *testing.T) {
		profile := &PublicProfile{NodeID: node, UserID: user, NodeAlias: "alice-laptop"}
		others, proof := receiveStatus(t, statusDatagram(t, signStatus(t, priv, false, profile)...))

		got, ok := astral.First[*PublicProfile](others)
		if !ok || !got.NodeID.IsEqual(node) || !got.UserID.IsEqual(user) {
			t.Fatalf("profile = %+v; want node %v and user %v", got, node, user)
		}
		hash, err := ProofHash(others, proof.Nonce, proof.Timestamp)
		if err != nil {
			t.Fatal(err)
		}
		if !verifies(t, got.NodeID.PublicKey(), hash, proof.Sig) {
			t.Error("the received visible proof does not verify")
		}
	})

	t.Run("stealth", func(t *testing.T) {
		hint := mustHint(t, node, user, astral.NewNonce())
		others, proof := receiveStatus(t, statusDatagram(t, signStatus(t, priv, true, hint)...))

		got, ok := astral.First[*StealthHint](others)
		if !ok {
			t.Fatal("no StealthHint received")
		}
		sender, ok := got.Open(user)
		if !ok || !sender.IsEqual(node) {
			t.Fatalf("Open(user) = %v, %v; want %v, true", sender, ok, node)
		}
		hash, err := ProofHash(others, proof.Nonce, proof.Timestamp)
		if err != nil {
			t.Fatal(err)
		}
		if verifies(t, sender.PublicKey(), hash, proof.Sig) {
			t.Error("the received stealth proof verifies before unwrapping")
		}
		if !verifies(t, sender.PublicKey(), hash, WrapProofSig(sender, proof.Nonce, proof.Sig)) {
			t.Error("the received stealth proof does not verify after unwrapping")
		}
	})
}
