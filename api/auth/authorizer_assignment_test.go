package auth

import (
	"bytes"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

func TestAuthorizerAssignmentRoundTrip(t *testing.T) {
	actor := astral.GenerateIdentity()

	for _, in := range []*AuthorizerAssignment{
		{Action: "mod.auth.see_objects_action", Endpoint: "astral://telepathy:auth.authorize"},
		{
			Action:     "mod.auth.see_objects_action",
			Actor:      actor,
			Endpoint:   "https://authz.example/authorize",
			Timeout:    astral.Duration(500 * time.Millisecond),
			TokenValue: "secret",
		},
	} {
		var buf bytes.Buffer
		if _, err := in.WriteTo(&buf); err != nil {
			t.Fatalf("write: %v", err)
		}

		out := &AuthorizerAssignment{}
		if _, err := out.ReadFrom(&buf); err != nil {
			t.Fatalf("read: %v", err)
		}

		if out.Action != in.Action || out.Endpoint != in.Endpoint || out.Timeout != in.Timeout || out.TokenValue != in.TokenValue {
			t.Fatalf("got %+v, want %+v", out, in)
		}
		if in.Actor == nil && out.Actor != nil && !out.Actor.IsZero() {
			t.Fatalf("action-wide assignment decoded with actor %v", out.Actor)
		}
		if in.Actor != nil && !out.Actor.IsEqual(in.Actor) {
			t.Fatalf("actor: got %v, want %v", out.Actor, in.Actor)
		}
	}
}

func TestAuthorizerAssignmentValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    AuthorizerAssignment
		ok   bool
	}{
		{"valid", AuthorizerAssignment{Action: "a", Endpoint: "astral://x:y"}, true},
		{"one token", AuthorizerAssignment{Action: "a", Endpoint: "http://x", TokenEnv: "T"}, true},
		{"no action", AuthorizerAssignment{Endpoint: "astral://x:y"}, false},
		{"no endpoint", AuthorizerAssignment{Action: "a"}, false},
		{"two tokens", AuthorizerAssignment{Action: "a", Endpoint: "http://x", TokenFile: "f", TokenValue: "v"}, false},
	} {
		if err := tc.a.Validate(); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v", tc.name, err)
		}
	}
}
