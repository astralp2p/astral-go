package messaging

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func sampleCredential() *IdentityCredential {
	return &IdentityCredential{
		Identity:  astral.GenerateIdentity(),
		Alias:     astral.String8("scout"),
		Token:     astral.String8("h4d8s2w6y1b9t3n7"),
		ExpiresAt: astral.Time(timeWithNanos()),
	}
}

func sampleIdentityInfo() *IdentityInfo {
	return &IdentityInfo{
		Identity: astral.GenerateIdentity(),
		Alias:    astral.String8("scout"),
	}
}

func TestIdentityCredential_BinaryRoundTrip(t *testing.T) {
	src := sampleCredential()

	dst, ok := binaryRoundTrip(t, src).(*IdentityCredential)
	if !ok {
		t.Fatal("decoded as another type")
	}
	if !dst.Identity.IsEqual(src.Identity) {
		t.Fatalf("identity: want %v, got %v", src.Identity, dst.Identity)
	}
	if dst.Alias != src.Alias || dst.Token != src.Token {
		t.Fatalf("alias/token: want %v/%v, got %v/%v", src.Alias, src.Token, dst.Alias, dst.Token)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}

// The credential is what MethodCreateIdentity answers under out=json, and the
// token must survive that path or the participant is minted unusable.
func TestIdentityCredential_JSONChannelRoundTrip(t *testing.T) {
	src := sampleCredential()

	dst, ok := jsonRoundTrip(t, src).(*IdentityCredential)
	if !ok {
		t.Fatal("received as another type")
	}
	if dst.Token != src.Token {
		t.Fatalf("token: want %v, got %v", src.Token, dst.Token)
	}
	if !dst.ExpiresAt.Time().Equal(src.ExpiresAt.Time()) {
		t.Fatalf("expiry: want %v, got %v", src.ExpiresAt.Time(), dst.ExpiresAt.Time())
	}
}

// The type split exists so a caller cannot read a withheld token as a
// participant that has none. IdentityInfo must therefore carry no token field
// at all — not an empty one — and no expiry, which belongs to a token.
func TestIdentityInfo_CarriesNoTokenField(t *testing.T) {
	fields := jsonFields(t, sampleIdentityInfo())

	for _, name := range []string{"Token", "ExpiresAt"} {
		if _, found := fields[name]; found {
			t.Fatalf("IdentityInfo carries a %v field: %v", name, fields)
		}
	}
	for _, name := range []string{"Identity", "Alias"} {
		if _, found := fields[name]; !found {
			t.Fatalf("IdentityInfo carries no %v field: %v", name, fields)
		}
	}
}
