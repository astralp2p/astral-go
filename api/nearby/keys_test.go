package nearby

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// testKey returns a deterministic key pair derived from seed, so a failure names the
// same inputs on every run.
func testKey(seed byte) (*secp256k1.PrivateKey, *astral.Identity) {
	priv := secp256k1.PrivKeyFromBytes(bytes.Repeat([]byte{seed}, 32))
	return priv, astral.IdentityFromPubKey(priv.PubKey())
}

// testIdentity returns the identity of testKey(seed).
func testIdentity(seed byte) *astral.Identity {
	_, id := testKey(seed)
	return id
}

// mustNotPanic runs fn and reports a panic as a test error.
func mustNotPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s panicked: %v", name, r)
		}
	}()
	fn()
}
