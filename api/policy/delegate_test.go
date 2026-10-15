package policy

import (
	"bytes"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

func TestDelegateRoundTrip(t *testing.T) {
	in := &Delegate{
		Identity:  astral.GenerateIdentity(),
		Operation: "control.decide_app_register",
		Timeout:   astral.Duration(5 * time.Minute),
	}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}

	out := &Delegate{}
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatalf("read: %v", err)
	}

	if !out.Identity.IsEqual(in.Identity) || out.Operation != in.Operation || out.Timeout != in.Timeout {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

// A nil delegate and one naming no identity both select the bundled policy.
func TestDelegateIsZero(t *testing.T) {
	var none *Delegate
	if !none.IsZero() || !(&Delegate{}).IsZero() || !(&Delegate{Operation: "x"}).IsZero() {
		t.Fatal("a delegate without an identity is not zero")
	}
	if (&Delegate{Identity: astral.GenerateIdentity()}).IsZero() {
		t.Fatal("a delegate with an identity is zero")
	}
}

func TestDelegateValidate(t *testing.T) {
	id := astral.GenerateIdentity()
	for _, tc := range []struct {
		name string
		d    Delegate
		ok   bool
	}{
		{"valid", Delegate{Identity: id, Operation: "op"}, true},
		{"at the bound", Delegate{Identity: id, Operation: "op", Timeout: astral.Duration(MaxTimeout)}, true},
		{"no identity", Delegate{Operation: "op"}, false},
		{"no operation", Delegate{Identity: id}, false},
		{"above the bound", Delegate{Identity: id, Operation: "op", Timeout: astral.Duration(MaxTimeout + time.Second)}, false},
	} {
		if err := tc.d.Validate(); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v", tc.name, err)
		}
	}
}

// A zero timeout waits the bound; a set one waits itself.
func TestDelegateTimeout(t *testing.T) {
	if got := (&Delegate{}).timeout(); got != MaxTimeout {
		t.Fatalf("zero timeout waits %v, want %v", got, MaxTimeout)
	}
	if got := (&Delegate{Timeout: astral.Duration(time.Second)}).timeout(); got != time.Second {
		t.Fatalf("set timeout waits %v, want 1s", got)
	}
}
