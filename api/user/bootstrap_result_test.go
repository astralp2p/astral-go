package user

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestBootstrapResult_BinaryRoundTrip(t *testing.T) {
	cases := []BootstrapResult{
		{Identity: astral.GenerateIdentity()},
		{Identity: astral.GenerateIdentity(), Mnemonic: "abandon abandon about"},
		{Identity: astral.GenerateIdentity(), Token: "b9c2e1a3d4f5867a"},
	}

	for _, orig := range cases {
		data, err := astral.EncodeBytes(&orig)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		got, err := astral.DecodeAs[*BootstrapResult](data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if !got.Identity.IsEqual(orig.Identity) || got.Mnemonic != orig.Mnemonic || got.Token != orig.Token {
			t.Fatalf("round trip = %+v; want %+v", got, orig)
		}
	}
}
