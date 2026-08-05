package astral

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"testing"
)

// The published vector for the untyped 5-byte payload "hello"
// (primitive-types/object_id.sha256.md).
const helloUntypedID = "data1km81js7f9cfdbauqoq3kash6f8o5naxfa878ejx8gbbuckjazgbr"

// An Untyped Object's Size and Hash cover the raw payload, with no Stamp and no type
// header (core-definitions/object-id.md). Two entry points over the same bytes must
// therefore agree.
func TestResolveObjectID_UntypedMatchesThePublishedVector(t *testing.T) {
	blob := Blob("hello")

	id, err := ResolveObjectID(&blob)
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	if id.Size != 5 {
		t.Errorf("want size 5 over the raw payload, got %d", id.Size)
	}
	if got := id.String(); got != helloUntypedID {
		t.Errorf("want the published vector\n  %s\ngot\n  %s", helloUntypedID, got)
	}
}

func TestResolveObjectID_UntypedAgreesWithResolve(t *testing.T) {
	for _, payload := range []string{"", "hello", strings.Repeat("x", 1000)} {
		blob := Blob(payload)

		viaObject, err := ResolveObjectID(&blob)
		if err != nil {
			t.Fatalf("ResolveObjectID(%d bytes): %v", len(payload), err)
		}

		viaStream, err := Resolve(bytes.NewReader([]byte(payload)))
		if err != nil {
			t.Fatalf("Resolve(%d bytes): %v", len(payload), err)
		}

		if !viaObject.IsEqual(viaStream) {
			t.Errorf("%d-byte payload resolves two ways:\n  object %s\n  stream %s",
				len(payload), viaObject, viaStream)
		}
	}
}

// The Object Hash of an Empty Object is the hash of an empty buffer.
func TestResolveObjectID_EmptyUntypedIsTheEmptyHash(t *testing.T) {
	var blob Blob

	id, err := ResolveObjectID(&blob)
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	if id.Size != 0 {
		t.Errorf("want size 0, got %d", id.Size)
	}
	if want := sha256.Sum256(nil); id.Hash != want {
		t.Errorf("want the empty-buffer hash, got % x", id.Hash)
	}
}

// A typed object keeps the Stamp and type header: this fix must not move any of them.
func TestResolveObjectID_TypedIsUnchanged(t *testing.T) {
	v := Uint16(7)

	id, err := ResolveObjectID(&v)
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	// Stamp (4) + string8 length prefix (1) + "uint16" (6) + payload (2)
	if id.Size != 13 {
		t.Errorf("want the canonical form hashed, size 13, got %d", id.Size)
	}
}

// Size is part of the id, so every serialisation must carry it and each round trip must
// return what it was given.
func TestObjectID_AllSerialisationsAgree(t *testing.T) {
	nonZeroHash := sha256.Sum256([]byte("x"))

	cases := map[string]ObjectID{
		"zero":            {},
		"size only":       {Size: 99},
		"hash only":       {Hash: nonZeroHash},
		"size and hash":   {Size: 99, Hash: nonZeroHash},
		"large size only": {Size: 1 << 40},
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := want.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo: %v", err)
			}
			var fromBinary ObjectID
			if _, err := fromBinary.ReadFrom(&buf); err != nil {
				t.Fatalf("ReadFrom: %v", err)
			}
			if !fromBinary.IsEqual(&want) {
				t.Errorf("binary round trip: want %s, got %s", &want, &fromBinary)
			}

			j, err := want.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			var fromJSON ObjectID
			if err := fromJSON.UnmarshalJSON(j); err != nil {
				t.Fatalf("UnmarshalJSON(%s): %v", j, err)
			}
			if !fromJSON.IsEqual(&want) {
				t.Errorf("json round trip: want %s, got %s", &want, &fromJSON)
			}

			txt, err := want.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText: %v", err)
			}
			var fromText ObjectID
			if err := fromText.UnmarshalText(txt); err != nil {
				t.Fatalf("UnmarshalText(%s): %v", txt, err)
			}
			if !fromText.IsEqual(&want) {
				t.Errorf("text round trip: want %s, got %s", &want, &fromText)
			}
		})
	}
}

func TestObjectID_IsEqualComparesSize(t *testing.T) {
	a := ObjectID{Size: 99}
	b := ObjectID{Size: 7}
	var zero ObjectID

	if a.IsEqual(&b) || b.IsEqual(&a) {
		t.Error("ids differing only in size must not compare equal")
	}
	if a.IsEqual(&zero) || zero.IsEqual(&a) {
		t.Error("a size-only id must not compare equal to the zero id")
	}
	if !zero.IsEqual(&ObjectID{}) {
		t.Error("the zero id must equal itself")
	}
}

// Peers built against the previous encoder emit "" for the zero id.
func TestObjectID_UnmarshalJSONAcceptsTheLegacyEmptyString(t *testing.T) {
	var id ObjectID

	if err := id.UnmarshalJSON([]byte(`""`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if !id.IsZero() {
		t.Errorf("want the zero id, got %s", &id)
	}
}
