package astral

import (
	"io"
	"strings"
	"testing"
)

// AllBlueprints returns the entries it could describe and aggregates the ones it could
// not. Two populations were mixed in that error: types that genuinely cannot be synced,
// and allowlisted primitives, which have no Blueprint by design and never needed one.
// The second is much the larger, so the aggregate read as noise and its one consumer
// discarded it — which is how an app type that cannot be described came to be dropped
// from the sync with no signal at all.

// undescribable holds a bare Go string, which the codec cannot assign a width to, so
// BlueprintOf fails on it. It is the shape an app author writes by accident.
type undescribable struct{ Name string }

func (undescribable) ObjectType() string                     { return "test.undescribable" }
func (u undescribable) WriteTo(w io.Writer) (int64, error)   { return Objectify(&u).WriteTo(w) }
func (u *undescribable) ReadFrom(r io.Reader) (int64, error) { return Objectify(u).ReadFrom(r) }

// notAStruct is a newtype over a primitive that does not declare PrimitiveAlias, so it is
// neither a struct nor an alias and BlueprintOf fails on it.
//
// It exists because bool no longer serves: bool is on the primitive allowlist, so its
// derivation failure is by design and is now suppressed. A test needing a genuine
// non-struct failure needs a name the allowlist does not hold.
type notAStruct Uint8

func (notAStruct) ObjectType() string                     { return "test.not_a_struct" }
func (n notAStruct) WriteTo(w io.Writer) (int64, error)   { return Uint8(n).WriteTo(w) }
func (n *notAStruct) ReadFrom(r io.Reader) (int64, error) { return (*Uint8)(n).ReadFrom(r) }

// An allowlisted primitive has no Blueprint by design, so reporting it as a derivation
// failure is a false positive — and there are enough of them to bury a true one.
func TestAllBlueprints_AllowlistedPrimitivesAreNotReportedAsFailures(t *testing.T) {
	_, err := DefaultBlueprints().AllBlueprints()
	if err == nil {
		return // nothing to check
	}

	for _, line := range strings.Split(err.Error(), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSuffix(fields[1], ":")
		if IsPrimitiveType(name) {
			t.Errorf("%q is an allowlisted primitive and has no Blueprint by design; "+
				"reporting it as a failure buries the entries that are real", name)
		}
	}
}

// The regression guard for how this was nearly broken. identity and time are allowlisted
// *and* derive a Blueprint, so suppressing by name before deriving silently drops them
// from the sync — the same class of loss this change exists to stop.
func TestAllBlueprints_KeepsAllowlistedPrimitivesThatDoDerive(t *testing.T) {
	all, _ := DefaultBlueprints().AllBlueprints()

	got := map[string]bool{}
	for _, b := range all {
		got[b.Type.String()] = true
	}

	for _, name := range []string{"identity", "time"} {
		if !IsPrimitiveType(name) {
			t.Fatalf("%q is expected to be allowlisted; the premise of this test moved", name)
		}
		if !got[name] {
			t.Errorf("%q derives a Blueprint and must still be returned: suppressing the "+
				"error must not suppress the entry", name)
		}
	}
}

// A type that genuinely cannot be described must still be reported, by name. This is the
// signal an app author needs and the one the noise was hiding.
func TestAllBlueprints_ReportsAnUndescribableType(t *testing.T) {
	bps := NewBlueprints(nil)
	if err := bps.Add(&undescribable{}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	all, err := bps.AllBlueprints()

	for _, b := range all {
		if b.Type.String() == "test.undescribable" {
			t.Fatal("a type with a bare Go string field must not derive a Blueprint")
		}
	}
	if err == nil {
		t.Fatal("want the undescribable type reported, got no error")
	}
	if !strings.Contains(err.Error(), "test.undescribable") {
		t.Errorf("want the error to name the type, got %v", err)
	}
}

// Suppression is scoped to the allowlist, not to derivation failure in general: a
// registry holding only an undescribable type still reports it.
func TestAllBlueprints_SuppressionIsScopedToTheAllowlist(t *testing.T) {
	bps := NewBlueprints(nil)
	if err := bps.Add(&undescribable{}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := bps.AllBlueprints()
	if err == nil {
		t.Fatal("an undescribable type is not an allowlisted primitive and must be reported")
	}
}
