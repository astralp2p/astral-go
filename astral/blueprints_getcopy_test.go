package astral

import "testing"

// RegisterBlueprint clones on the way in so a caller keeping its own copy cannot reach
// into the registry. GetBlueprint handed the stored pointer straight back out, which left
// the other half open: a caller could append a Field and change what every later decode
// of that type expects, from outside the registry's API.

func registerForCopyTest(t *testing.T) *Blueprints {
	t.Helper()

	bps := NewBlueprints(DefaultBlueprints())
	bp := NewBlueprint("test.copy_me", Field{
		Name: "A",
		Spec: &PrimitiveSpec{PrimitiveType: "uint8"},
	})
	if _, err := bps.RegisterBlueprint(bp); err != nil {
		t.Fatalf("RegisterBlueprint: %v", err)
	}
	return bps
}

func TestGetBlueprint_CallerCannotMutateTheRegistry(t *testing.T) {
	bps := registerForCopyTest(t)

	got := bps.GetBlueprint("test.copy_me")
	if got == nil {
		t.Fatal("want the registered blueprint")
	}
	got.Fields = append(got.Fields, Field{
		Name: "Injected",
		Spec: &PrimitiveSpec{PrimitiveType: "uint8"},
	})
	got.Fields[0].Name = "Renamed"

	again := bps.GetBlueprint("test.copy_me")
	if len(again.Fields) != 1 {
		t.Errorf("a caller appended a field and the registry kept it: want 1 field, got %d",
			len(again.Fields))
	}
	if again.Fields[0].Name.String() != "A" {
		t.Errorf("a caller renamed a field and the registry kept it: want %q, got %q",
			"A", again.Fields[0].Name)
	}
}

// Two callers must not share a descriptor either, or one can mutate the other's.
func TestGetBlueprint_ReturnsADistinctValuePerCall(t *testing.T) {
	bps := registerForCopyTest(t)

	a := bps.GetBlueprint("test.copy_me")
	b := bps.GetBlueprint("test.copy_me")

	if a == b {
		t.Fatal("want a distinct copy per call, got the same pointer")
	}
	a.Fields[0].Name = "Renamed"
	if b.Fields[0].Name.String() != "A" {
		t.Errorf("mutating one caller's copy changed another's: got %q", b.Fields[0].Name)
	}
}

// A missing name must still be nil rather than an empty blueprint: cloneBlueprint(nil)
// would otherwise turn "not registered" into "registered with no fields", which decodes
// very differently.
func TestGetBlueprint_MissingNameIsStillNil(t *testing.T) {
	bps := registerForCopyTest(t)

	if got := bps.GetBlueprint("test.not_registered"); got != nil {
		t.Errorf("want nil for an unregistered name, got %#v", got)
	}
}

// The existence check must not pay for a copy — it runs per element type during
// construction, the path narrowed to fix a DoS. This pins that it reads the stored
// pointer rather than cloning.
func TestGetBlueprintRef_DoesNotCopy(t *testing.T) {
	bps := registerForCopyTest(t)

	a := bps.getBlueprintRef("test.copy_me")
	b := bps.getBlueprintRef("test.copy_me")

	if a == nil {
		t.Fatal("want the stored blueprint")
	}
	if a != b {
		t.Error("the internal accessor must return the stored pointer, not a copy: " +
			"isRuntimeBlueprintType calls it per element type during construction")
	}
}
