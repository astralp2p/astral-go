package astral

import (
	"bytes"
	"testing"
)

// TestErrUnexpectedObject_Error pins Error naming the object's type, and naming the Nil
// marker for every absent form instead of panicking. An interface-nil Object panicked
// on a nil dereference; a typed nil pointer panicked calling a value-receiver
// ObjectType through it.
func TestErrUnexpectedObject_Error(t *testing.T) {
	cases := []struct {
		name   string
		object Object
		want   string
	}{
		{"object", &Ack{}, "unexpected object: ack"},
		{"interface nil", nil, "unexpected object: nil"},
		{"typed nil pointer", (*Ack)(nil), "unexpected object: nil"},
		{"nil marker", &Nil{}, "unexpected object: nil"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NewErrUnexpectedObject(c.object).Error(); got != c.want {
				t.Fatalf("Error: want %q, got %q", c.want, got)
			}
		})
	}
}

// TestErrUnexpectedObject_Error_AfterDecode pins the wire path: an encoded error with an
// absent Object decodes to a nil Object, and Error on the decoded value still returns.
func TestErrUnexpectedObject_Error_AfterDecode(t *testing.T) {
	var buf bytes.Buffer
	if _, err := NewErrUnexpectedObject(nil).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo returned error: %v", err)
	}

	var decoded ErrUnexpectedObject
	if _, err := decoded.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}
	if want, got := "unexpected object: nil", decoded.Error(); got != want {
		t.Fatalf("Error: want %q, got %q", want, got)
	}
}
