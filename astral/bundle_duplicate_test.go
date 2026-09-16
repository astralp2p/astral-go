package astral

import (
	"bytes"
	"errors"
	"testing"
)

// A Bundle's members are unique by Object ID, and all three ways into a Bundle enforce it.
// The rejection used to be an unnamed error, so a caller told it apart from a transport
// failure only by matching the message text. These tests pin the sentinel at each entry.

// duplicateString8Payload returns a bundle body whose two members are the same String8.
func duplicateString8Payload(t *testing.T, value String8) []byte {
	t.Helper()

	var payload bytes.Buffer
	if _, err := Uint32(2).WriteTo(&payload); err != nil {
		t.Fatalf("encode count: %v", err)
	}

	for range 2 {
		var member bytes.Buffer
		if _, err := String8("string8").WriteTo(&member); err != nil {
			t.Fatalf("encode member type: %v", err)
		}
		if _, err := value.WriteTo(&member); err != nil {
			t.Fatalf("encode member: %v", err)
		}
		if _, err := Bytes32(member.Bytes()).WriteTo(&payload); err != nil {
			t.Fatalf("encode member block: %v", err)
		}
	}

	return payload.Bytes()
}

func TestBundle_ReadFromRejectsDuplicateWithSentinel(t *testing.T) {
	alpha := String8("alpha")

	_, err := NewBundle().ReadFrom(bytes.NewReader(duplicateString8Payload(t, alpha)))

	if !errors.Is(err, ErrDuplicateObject) {
		t.Fatalf("want ErrDuplicateObject, got %v", err)
	}
}

func TestBundle_AppendRejectsDuplicateWithSentinel(t *testing.T) {
	alpha := String8("alpha")

	err := NewBundle().Append(&alpha, &alpha)

	if !errors.Is(err, ErrDuplicateObject) {
		t.Fatalf("want ErrDuplicateObject, got %v", err)
	}
}

func TestBundle_UnmarshalJSONRejectsDuplicateWithSentinel(t *testing.T) {
	const payload = `[{"Type":"string8","Object":"alpha"},{"Type":"string8","Object":"alpha"}]`

	err := NewBundle().UnmarshalJSON([]byte(payload))

	if !errors.Is(err, ErrDuplicateObject) {
		t.Fatalf("want ErrDuplicateObject, got %v", err)
	}
}

// The wrapped ID is what tells a caller which member repeated.
func TestBundle_DuplicateErrorNamesTheRepeatedObjectID(t *testing.T) {
	alpha := String8("alpha")

	objectID, err := ResolveObjectID(&alpha)
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	err = NewBundle().Append(&alpha, &alpha)

	if got := err.Error(); !bytes.Contains([]byte(got), []byte(objectID.String())) {
		t.Errorf("want the repeated object ID %v in %q", objectID, got)
	}
}
