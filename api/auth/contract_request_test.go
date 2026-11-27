package auth

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

func sampleRequestID() *astral.ObjectID {
	id := &astral.ObjectID{Size: 1234}
	copy(id.Hash[:], bytes.Repeat([]byte{0xab}, 32))
	return id
}

func sampleContract(requestID *astral.ObjectID) *Contract {
	return &Contract{
		Issuer:    astral.GenerateIdentity(),
		Subject:   astral.GenerateIdentity(),
		Permits:   []*Permit{{Action: "test.action", Delegation: 1}},
		ExpiresAt: astral.Time(time.Unix(1700000000, 5).UTC()),
		RequestID: requestID,
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
	case want.RequestID == nil && got.RequestID != nil:
		t.Fatalf("requestID: got %v, want nil", got.RequestID)
	case want.RequestID != nil && (got.RequestID == nil || *got.RequestID != *want.RequestID):
		t.Fatalf("requestID: got %v, want %v", got.RequestID, want.RequestID)
	}
	if !bytes.Equal(got.SignableHash(), want.SignableHash()) {
		t.Fatalf("signable hash changed across round trip")
	}
}

func TestContractRequestIDRoundTrip(t *testing.T) {
	for _, id := range []*astral.ObjectID{nil, sampleRequestID()} {
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

// TestSignedContractRequestIDRoundTrip covers the SwarmState form, half
// signed (an offer) and with both signatures.
func TestSignedContractRequestIDRoundTrip(t *testing.T) {
	sig := sampleSignature()
	for _, sc := range []*SignedContract{
		{Contract: sampleContract(sampleRequestID()), IssuerSig: sig},
		{Contract: sampleContract(sampleRequestID()), IssuerSig: sig, SubjectSig: sig},
		{Contract: sampleContract(nil), IssuerSig: sig, SubjectSig: sig},
	} {
		data, err := astral.EncodeBytes(sc)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		bin, err := astral.DecodeAs[*SignedContract](data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertContractEqual(t, sc.Contract, bin.Contract)
		if (bin.SubjectSig == nil) != (sc.SubjectSig == nil) || bin.IssuerSig == nil {
			t.Fatalf("signatures changed: %+v", bin)
		}

		j, err := sc.MarshalJSON()
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var js SignedContract
		if err := js.UnmarshalJSON(j); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		assertContractEqual(t, sc.Contract, js.Contract)
	}
}

// TestContractRequestIDChangesObjectID shows the link is bound by the
// contract's ObjectID, and so by both signatures over it.
func TestContractRequestIDChangesObjectID(t *testing.T) {
	c := sampleContract(nil)
	without := c.SignableHash()
	c.RequestID = sampleRequestID()
	if bytes.Equal(without, c.SignableHash()) {
		t.Fatal("RequestID does not change the signable hash")
	}
}

// legacyContract is auth.Contract as encoded before it carried a request link.
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
// RequestID: a contract encoded without the field fails to decode, and the
// same content hashes to a different ObjectID. New swarms only (D11).
func TestContractLegacyEncodingBreaks(t *testing.T) {
	c := sampleContract(nil)
	legacy := legacyContract{c.Issuer, c.Subject, c.Permits, c.ExpiresAt}

	var buf bytes.Buffer
	if _, err := legacy.WriteTo(&buf); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	var got Contract
	if _, err := got.ReadFrom(&buf); err == nil {
		t.Fatal("legacy contract decoded; update the compatibility note on Contract.RequestID")
	}

	legacyID, err := astral.ResolveObjectID(&legacy)
	if err != nil {
		t.Fatalf("legacy id: %v", err)
	}
	if bytes.Equal(legacyID.Hash[:], c.SignableHash()) {
		t.Fatal("legacy and new encodings hash alike; update the compatibility note")
	}
}

func sampleSignature() *crypto.Signature {
	return &crypto.Signature{Scheme: crypto.SchemeASN1, Data: astral.Bytes16("dummy-signature-bytes")}
}
