package astral

import (
	"io"
	"slices"
	"testing"
)

// Both listings feed the same three buckets and must agree on their order, because both
// describe the same replay: aliases first, then structs topologically sorted, so every
// reference edge targets an already-replayed name (topics/blueprints.md).
//
// A registry with no aliases cannot show the divergence — with nothing to misplace, the
// two orders coincide — which is why every case here registers one.

func TestBlueprintOrder_ListingsAgree(t *testing.T) {
	// why: the parent supplies uint8, which an alias's Underlying must resolve against.
	bps := NewBlueprints(DefaultBlueprints())

	alias := NewBlueprintAlias("aa.alias_mode", "uint8")
	if _, err := bps.RegisterBlueprint(alias); err != nil {
		t.Fatalf("RegisterBlueprint: %v", err)
	}

	names := bps.OrderedBlueprints()

	// why: the aggregate error names every registered type with no derivable Blueprint —
	// the primitives, by design — so it is expected here and not a failure.
	all, _ := bps.AllBlueprints()

	// project the name listing onto the set AllBlueprints describes: the two answer
	// different questions and differ in cardinality by design, but where they overlap
	// they must agree on sequence.
	described := map[string]bool{}
	var wantSeq []string
	for _, b := range all {
		described[b.Type.String()] = true
		wantSeq = append(wantSeq, b.Type.String())
	}

	var gotSeq []string
	for _, n := range names {
		if described[n] {
			gotSeq = append(gotSeq, n)
		}
	}

	if !slices.Equal(gotSeq, wantSeq) {
		t.Errorf("the two listings disagree on order:\n  OrderedBlueprints %v\n  AllBlueprints     %v",
			gotSeq, wantSeq)
	}
}

// An alias must precede any struct that references it, in both listings.
func TestBlueprintOrder_AliasPrecedesItsReferrer(t *testing.T) {
	bps := NewBlueprints(DefaultBlueprints())

	alias := NewBlueprintAlias("aa.mode", "uint8")
	if _, err := bps.RegisterBlueprint(alias); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	msg := NewBlueprint("zz.msg", Field{Name: "M", Spec: &RefSpec{Type: "aa.mode"}})
	if _, err := bps.RegisterBlueprint(msg); err != nil {
		t.Fatalf("register struct: %v", err)
	}

	names := bps.OrderedBlueprints()

	aliasAt := slices.Index(names, "aa.mode")
	msgAt := slices.Index(names, "zz.msg")

	if aliasAt < 0 || msgAt < 0 {
		t.Fatalf("both names must be listed, got %v", names)
	}
	if aliasAt > msgAt {
		t.Errorf("the alias must replay before the struct referencing it, got %v", names)
	}
}

// A struct must precede any struct that references it. astral.blueprint carries
// Fields []Field, which is exactly that shape.
func TestBlueprintOrder_ReferencedStructComesFirst(t *testing.T) {
	bps := NewBlueprints(nil)

	if err := bps.Add(&Blueprint{}, &Field{}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	names := bps.OrderedBlueprints()

	fieldAt := slices.Index(names, "astral.blueprint.field")
	blueprintAt := slices.Index(names, "astral.blueprint")

	if fieldAt < 0 || blueprintAt < 0 {
		t.Fatalf("both names must be listed, got %v", names)
	}
	if fieldAt > blueprintAt {
		t.Errorf("astral.blueprint references astral.blueprint.field, so the field type "+
			"must come first; got %v", names)
	}
}

// orderMode is a compile-time PrimitiveAlias prototype — a Go newtype over a primitive,
// registered with Add rather than RegisterBlueprint. It lands in the alias-prototype
// bucket, which is the one whose position relative to struct prototypes was wrong.
type orderMode Uint8

func (orderMode) ObjectType() string                     { return "aa.order_mode" }
func (orderMode) UnderlyingPrimitive() string            { return "uint8" }
func (m orderMode) WriteTo(w io.Writer) (int64, error)   { return Uint8(m).WriteTo(w) }
func (m *orderMode) ReadFrom(r io.Reader) (int64, error) { return (*Uint8)(m).ReadFrom(r) }

// orderMsg is a compile-time struct prototype referencing that alias.
type orderMsg struct {
	M orderMode
}

func (orderMsg) ObjectType() string                     { return "zz.order_msg" }
func (m orderMsg) WriteTo(w io.Writer) (int64, error)   { return Objectify(&m).WriteTo(w) }
func (m *orderMsg) ReadFrom(r io.Reader) (int64, error) { return Objectify(m).ReadFrom(r) }

// The case the two listings actually disagreed on: a compile-time alias prototype and a
// compile-time struct prototype that references it. Names are chosen so alphabetical
// order runs opposite to dependency order, which is what let an alpha sort look correct.
func TestBlueprintOrder_AliasPrototypePrecedesStructPrototype(t *testing.T) {
	bps := NewBlueprints(DefaultBlueprints())

	if err := bps.Add(new(orderMode), &orderMsg{}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	names := bps.OrderedBlueprints()

	aliasAt := slices.Index(names, "aa.order_mode")
	msgAt := slices.Index(names, "zz.order_msg")

	if aliasAt < 0 || msgAt < 0 {
		t.Fatalf("both names must be listed; got alias=%d msg=%d", aliasAt, msgAt)
	}
	if aliasAt > msgAt {
		t.Errorf("the alias prototype must replay before the struct referencing it, "+
			"got alias at %d and struct at %d", aliasAt, msgAt)
	}
}
