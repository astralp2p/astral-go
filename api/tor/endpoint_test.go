package tor

import (
	"encoding/json"
	"testing"
)

// UnmarshalText accepts exactly one text form for the zero endpoint, so
// MarshalText owes that form. Emitting .onion:0 instead made the type's own
// encoder produce text its own parser refuses.
func TestEndpointText_ZeroValueRoundTrips(t *testing.T) {
	text, err := (Endpoint{}).MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if want := "unknown"; string(text) != want {
		t.Errorf("MarshalText = %q, want %q", text, want)
	}

	var got Endpoint
	if err := got.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText(%q): %v", text, err)
	}
	if !got.IsZero() {
		t.Errorf("Endpoint = %v, want the zero endpoint", got.Address())
	}
}

// The zero value is the only case that changed; an endpoint carrying a digest
// keeps the <digest>:<port> form.
func TestEndpointText_RealValueRoundTrips(t *testing.T) {
	want := Endpoint{Digest: realDigest(), Port: 9001}

	text, err := want.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}

	var got Endpoint
	if err := got.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText(%q): %v", text, err)
	}
	if got.Address() != want.Address() {
		t.Errorf("Address = %q, want %q", got.Address(), want.Address())
	}
}

// MarshalJSON already delegated to Address; text now agrees with it rather
// than disagreeing on the zero value alone.
func TestEndpointText_AgreesWithJSON(t *testing.T) {
	for _, e := range []Endpoint{{}, {Digest: realDigest(), Port: 9001}} {
		text, err := e.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText: %v", err)
		}

		raw, err := json.Marshal(&e)
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}
		var asJSON string
		if err := json.Unmarshal(raw, &asJSON); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if string(text) != asJSON {
			t.Errorf("MarshalText = %q, JSON = %q", text, asJSON)
		}
	}
}
