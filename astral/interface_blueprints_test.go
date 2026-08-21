package astral

import (
	"bytes"
	"io"
	"testing"
)

// WithBlueprints exists so a caller can decode against types the default registry does
// not hold. Decode threads the registry into the reader wrapper and a struct's own
// fields resolve through it — but a polymorphic field resolved through the package-level
// New, which reads defaultBlueprints. That is the one slot whose type is not known until
// the bytes arrive, so it is exactly where the per-call registry has to be honoured.
//
// The symptom is shaped so it looks like a corrupt stream: the outer type resolves,
// because Decode looks that one up in cfg.Blueprints directly, and the first nested type
// fails with "blueprint not found".

// propNest carries a polymorphic field and is registered nowhere by default, so it can
// only be decoded through a per-call registry.
type propNest struct{ Inner Object }

func (propNest) ObjectType() string                     { return "test.prop_nest" }
func (p propNest) WriteTo(w io.Writer) (int64, error)   { return Objectify(&p).WriteTo(w) }
func (p *propNest) ReadFrom(r io.Reader) (int64, error) { return Objectify(p).ReadFrom(r) }

func propRegistry(t *testing.T) *Blueprints {
	t.Helper()

	bps := NewBlueprints(DefaultBlueprints())
	if err := bps.Add(&propNest{}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	return bps
}

// The defect. One level resolves because Decode looks the outer type up itself; two
// levels needs the registry to survive into the polymorphic field.
func TestWithBlueprints_ReachesAPolymorphicField(t *testing.T) {
	bps := propRegistry(t)

	inner := String8("x")
	obj := &propNest{Inner: &propNest{Inner: &inner}}

	var buf bytes.Buffer
	if _, err := Encode(&buf, obj); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, _, err := Decode(bytes.NewReader(buf.Bytes()), WithBlueprints(bps))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	outer, ok := got.(*propNest)
	if !ok {
		t.Fatalf("want *propNest, got %T", got)
	}
	mid, ok := outer.Inner.(*propNest)
	if !ok {
		t.Fatalf("want the nested field to decode as *propNest, got %T", outer.Inner)
	}
	if s, ok := mid.Inner.(*String8); !ok || string(*s) != "x" {
		t.Errorf("want the innermost value back, got %#v", mid.Inner)
	}
}

// The change must be strictly more permissive: a child registry walks its parent, so a
// type held only by the default registry still resolves in a polymorphic slot.
func TestWithBlueprints_StillResolvesThroughTheParent(t *testing.T) {
	bps := propRegistry(t)

	inner := String8("x") // registered in the default registry, not in bps
	var buf bytes.Buffer
	if _, err := Encode(&buf, &propNest{Inner: &inner}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, _, err := Decode(bytes.NewReader(buf.Bytes()), WithBlueprints(bps))
	if err != nil {
		t.Fatalf("a parent-registered type must still resolve in a polymorphic slot: %v", err)
	}
	if s, ok := got.(*propNest).Inner.(*String8); !ok || string(*s) != "x" {
		t.Errorf("want the parent-registered value back, got %#v", got.(*propNest).Inner)
	}
}

// No registry threaded: resolve() falls back to defaultBlueprints, so the default path
// is untouched.
func TestWithBlueprints_DefaultPathUnchanged(t *testing.T) {
	inner := String8("x")
	obj := &ErrUnexpectedObject{Object: &inner} // registered in the default registry

	var buf bytes.Buffer
	if _, err := Encode(&buf, obj); err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, _, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode with no per-call registry: %v", err)
	}
	if s, ok := got.(*ErrUnexpectedObject).Object.(*String8); !ok || string(*s) != "x" {
		t.Errorf("want the value back, got %#v", got.(*ErrUnexpectedObject).Object)
	}
}
