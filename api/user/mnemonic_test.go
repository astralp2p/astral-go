package user

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestMnemonic_BinaryRoundTrip(t *testing.T) {
	orig := Mnemonic{Words: "abandon abandon about"}

	data, err := astral.EncodeBytes(&orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := astral.DecodeAs[*Mnemonic](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.Words != orig.Words {
		t.Fatalf("round trip = %+v; want %+v", got, orig)
	}
}
