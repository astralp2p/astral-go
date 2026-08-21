package astral

import (
	"bytes"
	"errors"
	"testing"
)

// Decoding is driven entirely by the peer. The guard that bounds it is only as good as
// its weakest frame, so the tests below are about the guard as a whole rather than about
// any one nesting shape: unbounded depth, a laundered counter and an unbounded fan-out
// are the same failure seen from three sides.

// bundlePayload returns the body of a bundle carrying one member of the given type and
// payload. A zero count is the innermost bundle.
func bundlePayload(t *testing.T, memberType string, memberPayload []byte) []byte {
	t.Helper()

	var member bytes.Buffer
	if _, err := String8(memberType).WriteTo(&member); err != nil {
		t.Fatalf("encode member type: %v", err)
	}
	member.Write(memberPayload)

	var out bytes.Buffer
	if _, err := Uint32(1).WriteTo(&out); err != nil {
		t.Fatalf("encode count: %v", err)
	}
	if _, err := Bytes32(member.Bytes()).WriteTo(&out); err != nil {
		t.Fatalf("encode member block: %v", err)
	}
	return out.Bytes()
}

// nestedBundles builds a stream of depth bundles, each holding the next.
func nestedBundles(t *testing.T, depth int) []byte {
	t.Helper()

	var empty bytes.Buffer
	if _, err := Uint32(0).WriteTo(&empty); err != nil {
		t.Fatalf("encode empty bundle: %v", err)
	}
	payload := empty.Bytes()

	for range depth {
		payload = bundlePayload(t, "bundle", payload)
	}

	var out bytes.Buffer
	if _, err := String8("bundle").WriteTo(&out); err != nil {
		t.Fatalf("encode outer type: %v", err)
	}
	out.Write(payload)
	return out.Bytes()
}

// A Bundle used to hand each member to a bare reader, which started a fresh counter. One
// Bundle anywhere in a payload therefore reset the depth guard for everything below it.
func TestDecode_NestedBundlesAreBounded(t *testing.T) {
	_, _, err := Decode(bytes.NewReader(nestedBundles(t, 400)))

	if !errors.Is(err, ErrDepthExceeded) {
		t.Fatalf("want ErrDepthExceeded for 400 nested bundles, got %v", err)
	}
}

// The cap must still admit what it is set to admit.
func TestDecode_ShallowBundlesStillDecode(t *testing.T) {
	o, _, err := Decode(bytes.NewReader(nestedBundles(t, 4)))

	if err != nil {
		t.Fatalf("want a shallow bundle to decode, got %v", err)
	}
	if _, ok := o.(*Bundle); !ok {
		t.Fatalf("want a *Bundle, got %T", o)
	}
}

// The distinguishing claim. With a laundered guard no nesting depth ever fails, because
// each Bundle restarts the counter; with a shared one the first failure arrives at the
// cap. Searching for the flip point asserts the guard engages at all, and asserts it
// engages in the right place.
func TestDecode_BundleNestingFailsAtTheCap(t *testing.T) {
	firstFailure := -1

	for depth := 1; depth <= 4*MaxBlueprintDepth; depth++ {
		if _, _, err := Decode(bytes.NewReader(nestedBundles(t, depth))); err != nil {
			if !errors.Is(err, ErrDepthExceeded) {
				t.Fatalf("depth %d: want ErrDepthExceeded, got %v", depth, err)
			}
			firstFailure = depth
			break
		}
	}

	if firstFailure < 0 {
		t.Fatalf("no nesting depth up to %d was refused; the guard is being reset per frame",
			4*MaxBlueprintDepth)
	}
	if firstFailure > MaxBlueprintDepth {
		t.Errorf("first refusal at depth %d, past the cap of %d",
			firstFailure, MaxBlueprintDepth)
	}
}

// A duplicate member is rejected, and the receiver keeps what it had.
func TestBundle_ReadFromLeavesTheReceiverIntactOnFailure(t *testing.T) {
	alpha := String8("alpha")

	original := NewBundle()
	if err := original.Append(&alpha); err != nil {
		t.Fatalf("Append: %v", err)
	}

	var payload bytes.Buffer
	if _, err := Uint32(2).WriteTo(&payload); err != nil {
		t.Fatalf("encode count: %v", err)
	}
	for range 2 {
		var member bytes.Buffer
		if _, err := String8("string8").WriteTo(&member); err != nil {
			t.Fatalf("encode member type: %v", err)
		}
		if _, err := alpha.WriteTo(&member); err != nil {
			t.Fatalf("encode member: %v", err)
		}
		if _, err := Bytes32(member.Bytes()).WriteTo(&payload); err != nil {
			t.Fatalf("encode member block: %v", err)
		}
	}

	err := errors.New("")
	_, err = original.ReadFrom(bytes.NewReader(payload.Bytes()))

	if err == nil {
		t.Fatal("want an error for a duplicated member")
	}
	if got := len(original.Objects()); got != 1 {
		t.Errorf("want the receiver's previous contents intact, got %d objects", got)
	}
}
