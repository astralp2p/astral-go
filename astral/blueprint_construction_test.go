package astral

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"
)

// Blueprint registration is reachable by a peer, so constructing a value from a
// registered Blueprint must cost work bounded by the registry — not by the shape a
// peer chose. Neither test below needs a cycle: every registration here is acyclic
// and passes validateReferences.

// buildBudgetDeadline bounds a construction that is supposed to be cheap. The
// unfixed code takes tens of seconds at these sizes, so the margin is wide enough
// not to flake and narrow enough to catch a return to exponential cost.
const buildBudgetDeadline = 5 * time.Second

// chain registers depth Blueprints where each level references the one below it
// through spec, with fanout fields per level.
func chain(t *testing.T, bps *Blueprints, prefix string, depth, fanout int, spec func(below string) Spec) string {
	t.Helper()

	below := prefix + "0"
	if _, err := bps.RegisterBlueprint(NewBlueprint(below, Field{Name: "V", Spec: &PrimitiveSpec{PrimitiveType: "uint8"}})); err != nil {
		t.Fatalf("register %s: %v", below, err)
	}

	for i := 1; i <= depth; i++ {
		name := fmt.Sprintf("%s%d", prefix, i)

		fields := make([]Field, 0, fanout)
		for f := range fanout {
			fields = append(fields, Field{Name: String16(fmt.Sprintf("F%d", f)), Spec: spec(below)})
		}

		if _, err := bps.RegisterBlueprint(NewBlueprint(name, fields...)); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
		below = name
	}

	return below
}

// mustBuildWithin fails if constructing typeName outruns the deadline. The build runs
// on its own goroutine so a runaway construction fails this test rather than hanging
// the package until the go test timeout fires.
func mustBuildWithin(t *testing.T, bps *Blueprints, typeName string, d time.Duration) Object {
	t.Helper()

	type result struct{ obj Object }
	done := make(chan result, 1)

	go func() { done <- result{bps.New(typeName)} }()

	select {
	case r := <-done:
		return r.obj
	case <-time.After(d):
		t.Fatalf("constructing %s did not finish within %v", typeName, d)
		return nil
	}
}

// A slice element naming a registered Blueprint used to rebuild that Blueprint's whole
// prototype, restarting the construction depth each time.
func TestNew_SliceChainIsBounded(t *testing.T) {
	bps := NewBlueprints(DefaultBlueprints())

	top := chain(t, bps, "slicechain.l", 30, 2, func(below string) Spec {
		return &SliceSpec{Type: String16(below)}
	})

	if o := mustBuildWithin(t, bps, top, buildBudgetDeadline); o == nil {
		t.Fatalf("want %s to construct", top)
	}
}

// A reference chain recurses by design and is capped by depth, but depth does not bound
// the width: k fields per level cost k^depth frames well inside MaxBlueprintDepth.
func TestNew_ReferenceFanOutIsBounded(t *testing.T) {
	bps := NewBlueprints(DefaultBlueprints())

	top := chain(t, bps, "refchain.l", 40, 2, func(below string) Spec {
		return &RefSpec{Type: String16(below)}
	})

	// The budget refuses this one; refusing is the point. It must refuse promptly and
	// through New's ordinary nil return rather than by exhausting the machine.
	mustBuildWithin(t, bps, top, buildBudgetDeadline)
}

// The budget surfaces as a typed error on the path that reports one.
func TestNewRuntimeObject_ReferenceFanOutExceedsTheBudget(t *testing.T) {
	bps := NewBlueprints(DefaultBlueprints())

	top := chain(t, bps, "refbudget.l", 40, 2, func(below string) Spec {
		return &RefSpec{Type: String16(below)}
	})

	_, err := newRuntimeObjectAt(bps, bps.GetBlueprint(top), 0, newBuildBudget())

	if !errors.Is(err, ErrDepthExceeded) {
		t.Fatalf("want ErrDepthExceeded, got %v", err)
	}
}

// A container that names its own type used to overflow the stack at construction:
// building the element prototype re-entered the same Blueprint, and bps.New restarted
// the depth at zero on every hop.
//
// RegisterBlueprint refuses this shape today — validateReferences requires the
// referenced name to be registered already, so the closing edge always fails — and the
// entries bypass is how the existing depth tests install one. Pinning it here keeps the
// overflow from returning if that validation is ever relaxed to admit tree schemas,
// which the wire itself handles: the element count bounds each frame.
func TestNew_SelfReferentialContainerDoesNotOverflow(t *testing.T) {
	// why: the decode path resolves a slice element through the registry carried on the
	// reader, which falls back to the default. The existing depth tests install their
	// cycles here for the same reason.
	bps := DefaultBlueprints()

	const node = "test.tree.node"
	bp := NewBlueprint(node,
		Field{Name: "Name", Spec: &PrimitiveSpec{PrimitiveType: "string16"}},
		Field{Name: "Kids", Spec: &SliceSpec{Type: node}},
	)
	bps.entries.Set(node, bp)

	o := mustBuildWithin(t, bps, node, buildBudgetDeadline)
	if o == nil {
		t.Fatalf("want %s to construct", node)
	}

	var buf bytes.Buffer
	if _, err := o.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	got := mustBuildWithin(t, bps, node, buildBudgetDeadline)
	if _, err := got.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
}

// The budget is per-construction, not per-registry: an ordinary Blueprint keeps
// building however many times it is asked for.
func TestNew_BudgetDoesNotLeakBetweenConstructions(t *testing.T) {
	bps := NewBlueprints(DefaultBlueprints())

	top := chain(t, bps, "reuse.l", 6, 2, func(below string) Spec {
		return &RefSpec{Type: String16(below)}
	})

	for i := range 200 {
		if o := bps.New(top); o == nil {
			t.Fatalf("construction %d returned nil; the budget is leaking across calls", i)
		}
	}
}
