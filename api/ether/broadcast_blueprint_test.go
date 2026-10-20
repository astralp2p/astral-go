package ether

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestBroadcastBlueprintDescribesTheWire(t *testing.T) {
	bp, err := astral.BlueprintOf(&Broadcast{})
	if err != nil {
		t.Fatal(err)
	}

	want := []struct{ name, primitive string }{
		{"Timestamp", "time"},
		{"Source", "identity"},
		{"Object", "bytes16"},
	}
	if len(bp.Fields) != len(want) {
		t.Fatalf("Fields = %d; want %d", len(bp.Fields), len(want))
	}
	for i, w := range want {
		f := bp.Fields[i]
		spec, ok := f.Spec.(*astral.PrimitiveSpec)
		if f.Name.String() != w.name || !ok || spec.PrimitiveType.String() != w.primitive {
			t.Errorf("Fields[%d] = %s %#v; want %s PrimitiveSpec %s", i, f.Name, f.Spec, w.name, w.primitive)
		}
	}
}

// A peer with no Go Broadcast decodes the bytes WriteTo writes from the Blueprint alone and
// writes them back unchanged.
func TestBroadcastBlueprintDecodesTheWireBytes(t *testing.T) {
	bp, err := astral.BlueprintOf(&Broadcast{})
	if err != nil {
		t.Fatal(err)
	}

	peer := astral.NewBlueprints(nil)
	for _, name := range []string{"time", "identity", "bytes16"} {
		if err := peer.Add(astral.New(name)); err != nil {
			t.Fatalf("primitive %s: %v", name, err)
		}
	}
	if _, err := peer.RegisterBlueprint(bp); err != nil {
		t.Fatal(err)
	}

	b := newTestBroadcast(t)
	var wire bytes.Buffer
	if _, err := b.WriteTo(&wire); err != nil {
		t.Fatal(err)
	}

	obj := peer.New(b.ObjectType())
	if _, ok := obj.(*astral.RuntimeObject); !ok {
		t.Fatalf("peer.New = %T; want *astral.RuntimeObject", obj)
	}
	read, err := obj.ReadFrom(bytes.NewReader(wire.Bytes()))
	if err != nil {
		t.Fatalf("read by Blueprint: %v", err)
	}
	if read != int64(wire.Len()) {
		t.Errorf("read %d bytes; wire holds %d", read, wire.Len())
	}

	var back bytes.Buffer
	if _, err := obj.WriteTo(&back); err != nil {
		t.Fatalf("write by Blueprint: %v", err)
	}
	if !bytes.Equal(back.Bytes(), wire.Bytes()) {
		t.Errorf("Blueprint bytes differ:\n want %s\n  got %s",
			hex.EncodeToString(wire.Bytes()), hex.EncodeToString(back.Bytes()))
	}
}

// SignedBroadcast inherits DeclareBlueprint from the embedded Broadcast; its own Blueprint
// still names itself and references mod.ether.broadcast.
func TestSignedBroadcastBlueprintNamesItself(t *testing.T) {
	bp, err := astral.BlueprintOf(&SignedBroadcast{})
	if err != nil {
		t.Fatal(err)
	}
	if bp.Type.String() != (SignedBroadcast{}).ObjectType() {
		t.Fatalf("Type = %s; want %s", bp.Type, (SignedBroadcast{}).ObjectType())
	}
	ref, ok := bp.Fields[0].Spec.(*astral.RefSpec)
	if !ok || ref.Type.String() != (Broadcast{}).ObjectType() {
		t.Fatalf("Broadcast spec = %#v; want RefSpec %s", bp.Fields[0].Spec, (Broadcast{}).ObjectType())
	}
}
