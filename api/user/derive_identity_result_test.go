package user

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestDeriveIdentityResult_BinaryRoundTrip(t *testing.T) {
	cases := []DeriveIdentityResult{
		{Identity: astral.GenerateIdentity()},
		{Identity: astral.GenerateIdentity(), Mnemonic: "abandon abandon about"},
	}

	for _, orig := range cases {
		data, err := astral.EncodeBytes(&orig)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		got, err := astral.DecodeAs[*DeriveIdentityResult](data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if !got.Identity.IsEqual(orig.Identity) || got.Mnemonic != orig.Mnemonic {
			t.Fatalf("round trip = %+v; want %+v", got, orig)
		}
	}
}
