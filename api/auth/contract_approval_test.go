package auth

import (
	"bytes"
	"encoding/base64"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

func sampleApprovalID() *astral.ObjectID {
	id := &astral.ObjectID{Size: 1234}
	copy(id.Hash[:], bytes.Repeat([]byte{0xab}, 32))
	return id
}

func sampleContract(approvalID *astral.ObjectID) *Contract {
	return &Contract{
		Issuer:     astral.GenerateIdentity(),
		Subject:    astral.GenerateIdentity(),
		Permits:    []*Permit{{Action: "test.action", Delegation: 1}},
		ExpiresAt:  astral.Time(time.Unix(1700000000, 5).UTC()),
		ApprovalID: approvalID,
	}
}

func assertContractEqual(t *testing.T, want, got *Contract) {
	t.Helper()
	if !got.Issuer.IsEqual(want.Issuer) || !got.Subject.IsEqual(want.Subject) {
		t.Fatalf("identities changed")
	}
	if len(got.Permits) != len(want.Permits) || got.Permits[0].Action != want.Permits[0].Action {
		t.Fatalf("permits changed: %+v", got.Permits)
	}
	if !got.ExpiresAt.Time().Equal(want.ExpiresAt.Time()) {
		t.Fatalf("expiresAt changed")
	}
	switch {
	case want.ApprovalID == nil && got.ApprovalID != nil:
		t.Fatalf("approvalID: got %v, want nil", got.ApprovalID)
	case want.ApprovalID != nil && (got.ApprovalID == nil || *got.ApprovalID != *want.ApprovalID):
		t.Fatalf("approvalID: got %v, want %v", got.ApprovalID, want.ApprovalID)
	}
	if !bytes.Equal(got.SignableHash(), want.SignableHash()) {
		t.Fatalf("signable hash changed across round trip")
	}
}

func TestContractApprovalIDRoundTrip(t *testing.T) {
	for _, id := range []*astral.ObjectID{nil, sampleApprovalID()} {
		orig := sampleContract(id)

		data, err := astral.EncodeBytes(orig)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		bin, err := astral.DecodeAs[*Contract](data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertContractEqual(t, orig, bin)

		j, err := orig.MarshalJSON()
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var js Contract
		if err := js.UnmarshalJSON(j); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		assertContractEqual(t, orig, &js)
	}
}

// TestContractApprovalIDChangesObjectID shows the link is bound by the
// contract's ObjectID, and so by the signature over it.
func TestContractApprovalIDChangesObjectID(t *testing.T) {
	c := sampleContract(nil)
	without := c.SignableHash()
	c.ApprovalID = sampleApprovalID()
	if bytes.Equal(without, c.SignableHash()) {
		t.Fatal("ApprovalID does not change the signable hash")
	}
}

// legacyContract is auth.Contract as encoded before ApprovalID existed.
type legacyContract struct {
	Issuer    *astral.Identity
	Subject   *astral.Identity
	Permits   []*Permit
	ExpiresAt astral.Time
}

func (legacyContract) ObjectType() string { return "mod.auth.contract" }
func (c legacyContract) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&c).WriteTo(w)
}
func (c *legacyContract) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(c).ReadFrom(r) }

// TestContractLegacyEncodingBreaks records the compatibility impact of
// ApprovalID: a contract encoded before the field existed fails to decode,
// and the same content hashes to a different ObjectID. New swarms only (D11).
func TestContractLegacyEncodingBreaks(t *testing.T) {
	c := sampleContract(nil)
	legacy := legacyContract{c.Issuer, c.Subject, c.Permits, c.ExpiresAt}

	var buf bytes.Buffer
	if _, err := legacy.WriteTo(&buf); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	var got Contract
	if _, err := got.ReadFrom(&buf); err == nil {
		t.Fatal("legacy contract decoded; update the compatibility note on Contract.ApprovalID")
	}

	legacyID, err := astral.ResolveObjectID(&legacy)
	if err != nil {
		t.Fatalf("legacy id: %v", err)
	}
	if bytes.Equal(legacyID.Hash[:], c.SignableHash()) {
		t.Fatal("legacy and new encodings hash alike; update the compatibility note")
	}
}

// signableLine is the text a text signer such as a Coldcard signs: the 15-byte
// commitment in base64, then SignableText (astrald mod/crypto formatSignableText).
func signableLine(hash []byte, text string) string {
	return "[" + base64.StdEncoding.EncodeToString(hash[0:15]) + "] " + text
}

func TestContractSignableTextWorstCase(t *testing.T) {
	c := sampleContract(&astral.ObjectID{Size: math.MaxUint64})
	c.ExpiresAt = astral.Time(time.Unix(0, math.MaxInt64))
	word := strings.Repeat("z", MaxPermitWordLen)

	line := signableLine(c.SignableHash(), contractText(c, word, math.MaxUint32))
	if len(line) > 240 {
		t.Fatalf("worst case is %d bytes, over 240: %q", len(line), line)
	}
	for i := 0; i < len(line); i++ {
		if line[i] < 0x20 || line[i] > 0x7e {
			t.Fatalf("byte %d is not printable ASCII: %q", i, line)
		}
	}
	t.Logf("contract worst case: %d bytes", len(line))
}

func TestValidPermitWord(t *testing.T) {
	for w, want := range map[string]bool{
		"privileged_member":      true,
		"member":                 true,
		"":                       false,
		"Member":                 false,
		"privileged_member_plus": false,
		"tier 1":                 false,
	} {
		if got := validPermitWord(w); got != want {
			t.Errorf("validPermitWord(%q) = %v, want %v", w, got, want)
		}
	}
}
