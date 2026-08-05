package user

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// sampleOpUpdate carries a non-zero value in every field, so a field dropped or
// mis-sized on the wire cannot pass as the zero value it decodes to.
func sampleOpUpdate() *OpUpdate {
	return &OpUpdate{
		Nonce:    astral.Nonce(0x0123456789abcdef),
		ObjectID: &astral.ObjectID{Size: 4096, Hash: [32]byte{1, 2, 3, 31: 255}},
		Removed:  true,
	}
}

func TestOpUpdate_BinaryRoundTrip(t *testing.T) {
	orig := sampleOpUpdate()

	data, err := astral.EncodeBytes(orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := astral.DecodeAs[*OpUpdate](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.Nonce != orig.Nonce {
		t.Errorf("nonce mismatch: got %v want %v", got.Nonce, orig.Nonce)
	}
	if !got.ObjectID.IsEqual(orig.ObjectID) {
		t.Errorf("objectID mismatch: got %v want %v", got.ObjectID, orig.ObjectID)
	}
	if got.Removed != orig.Removed {
		t.Errorf("removed mismatch: got %v want %v", got.Removed, orig.Removed)
	}
}

// TestOpUpdate_Registered confirms the type is registered with astral, which is
// what lets a sync_assets stream decode its entries by type name.
func TestOpUpdate_Registered(t *testing.T) {
	if obj := astral.New("mod.user.op_update"); obj == nil {
		t.Fatal(`type "mod.user.op_update" not registered`)
	}
}
