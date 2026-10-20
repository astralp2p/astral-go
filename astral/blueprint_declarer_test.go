package astral

import (
	"errors"
	"io"
	"testing"
)

// DeclaringWitness writes a bare identity by hand. Its Go field is a pointer, which reflection
// derives as a PtrSpec with a presence flag the wire does not carry.
type DeclaringWitness struct{ ID *Identity }

// astral:blueprint-ignore
func (DeclaringWitness) ObjectType() string { return "test.declarer.witness" }
func (w DeclaringWitness) WriteTo(dst io.Writer) (int64, error) {
	return w.ID.WriteTo(dst)
}
func (w *DeclaringWitness) ReadFrom(r io.Reader) (int64, error) {
	w.ID = &Identity{}
	return w.ID.ReadFrom(r)
}
func (DeclaringWitness) DeclareBlueprint() *Blueprint {
	return NewBlueprint("test.declarer.witness",
		Field{Name: "ID", Spec: &PrimitiveSpec{PrimitiveType: "identity"}})
}

// embeddingWitness inherits DeclaringWitness.DeclareBlueprint through embedding.
type embeddingWitness struct {
	DeclaringWitness
	N Uint8
}

// astral:blueprint-ignore
func (embeddingWitness) ObjectType() string { return "test.declarer.embedding" }
func (w embeddingWitness) WriteTo(dst io.Writer) (int64, error) {
	return Objectify(&w).WriteTo(dst)
}
func (w *embeddingWitness) ReadFrom(r io.Reader) (int64, error) { return Objectify(w).ReadFrom(r) }

// invalidDeclarer declares a primitive outside the allowlist.
type invalidDeclarer struct{ X Uint8 }

// astral:blueprint-ignore
func (invalidDeclarer) ObjectType() string                     { return "test.declarer.invalid" }
func (w invalidDeclarer) WriteTo(dst io.Writer) (int64, error) { return Objectify(&w).WriteTo(dst) }
func (w *invalidDeclarer) ReadFrom(r io.Reader) (int64, error) { return Objectify(w).ReadFrom(r) }
func (invalidDeclarer) DeclareBlueprint() *Blueprint {
	return NewBlueprint("test.declarer.invalid",
		Field{Name: "X", Spec: &PrimitiveSpec{PrimitiveType: "uint24"}})
}

func TestBlueprintOf_ReturnsTheDeclaredBlueprint(t *testing.T) {
	bp, err := BlueprintOf(&DeclaringWitness{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bp.Fields) != 1 {
		t.Fatalf("Fields = %d; want 1", len(bp.Fields))
	}
	spec, ok := bp.Fields[0].Spec.(*PrimitiveSpec)
	if !ok || spec.PrimitiveType != "identity" {
		t.Fatalf("ID spec = %#v; want PrimitiveSpec identity", bp.Fields[0].Spec)
	}
}

func TestBlueprintOf_DeclaredBlueprintIsACopy(t *testing.T) {
	bp, err := BlueprintOf(&DeclaringWitness{})
	if err != nil {
		t.Fatal(err)
	}
	bp.Fields[0].Name = "Changed"

	again, err := BlueprintOf(&DeclaringWitness{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Fields[0].Name != "ID" {
		t.Fatalf("Name = %q after mutating an earlier result; want ID", again.Fields[0].Name)
	}
}

func TestBlueprintOf_PromotedDeclarationIsIgnored(t *testing.T) {
	bp, err := BlueprintOf(&embeddingWitness{})
	if err != nil {
		t.Fatal(err)
	}
	if bp.Type != "test.declarer.embedding" {
		t.Fatalf("Type = %q; want test.declarer.embedding", bp.Type)
	}
	if len(bp.Fields) != 2 {
		t.Fatalf("Fields = %d; want 2 derived by reflection", len(bp.Fields))
	}
	ref, ok := bp.Fields[0].Spec.(*RefSpec)
	if !ok || ref.Type != "test.declarer.witness" {
		t.Fatalf("embedded spec = %#v; want RefSpec test.declarer.witness", bp.Fields[0].Spec)
	}
}

func TestBlueprintOf_InvalidDeclarationFails(t *testing.T) {
	_, err := BlueprintOf(&invalidDeclarer{})
	if !errors.Is(err, ErrBlueprintInvalid) {
		t.Fatalf("err = %v; want ErrBlueprintInvalid", err)
	}
}
