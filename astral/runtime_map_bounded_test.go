package astral

import (
	"bytes"
	"testing"
)

// A RuntimeMap whose value type is a runtime Blueprint decodes entries one by one. Its
// count is the peer's claim: sizing the map to it let a four-byte count name an
// arbitrarily large allocation before any entry arrived.
func TestRuntimeMap_ReadFromDoesNotReserveUnarrivedEntries(t *testing.T) {
	entryBP := NewBlueprint("test.bounded.map.entry",
		Field{Name: "x", Spec: &PrimitiveSpec{PrimitiveType: "uint32"}},
	)
	if _, err := Register(entryBP); err != nil {
		t.Fatal(err)
	}

	// one complete entry follows the count: key "a", presence flag 1, x = 7
	entry := []byte{0, 1, 'a', 1, 0, 0, 0, 7}

	for _, count := range []uint32{1 << 20, 1<<32 - 1} {
		m, err := NewRuntimeMap("string16", "test.bounded.map.entry")
		if err != nil {
			t.Fatal(err)
		}
		payload := claim(count, entry)

		used := allocatedBy(func() {
			_, err = m.ReadFrom(bytes.NewReader(payload))
		})

		if err == nil {
			t.Errorf("count %d: want an error for a count larger than the entries delivered", count)
		}
		if used > allocBudget {
			t.Errorf("count %d: allocated %d bytes for a %d-byte payload", count, used, len(payload))
		}
	}
}
