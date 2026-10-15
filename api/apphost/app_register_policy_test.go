package apphost

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// A decision carrying an assignment round-trips with the assignment intact
// and no actor named.
func TestAppRegisterDecisionRoundTripsAuthorizers(t *testing.T) {
	in := &AppRegisterDecision{
		Allow: true,
		Authorizers: []*auth.AuthorizerAssignment{{
			Action:   "mod.auth.see_objects_action",
			Endpoint: "astral://control-plane:control.authorize",
		}},
	}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := &AppRegisterDecision{}
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatalf("read: %v", err)
	}

	if !out.Allow || len(out.Authorizers) != 1 {
		t.Fatalf("got %+v, want allow with one assignment", out)
	}
	a := out.Authorizers[0]
	if a.Action != in.Authorizers[0].Action || a.Endpoint != in.Authorizers[0].Endpoint {
		t.Fatalf("assignment arrived as %+v", a)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("a decoded decision without an actor failed validation: %v", err)
	}
}

func TestAppRegisterDecisionValidate(t *testing.T) {
	actor := astral.GenerateIdentity()
	valid := &auth.AuthorizerAssignment{Action: "a", Endpoint: "astral://x:y"}

	for _, tc := range []struct {
		name string
		d    AppRegisterDecision
		ok   bool
	}{
		{"no assignments", AppRegisterDecision{Allow: true}, true},
		{"valid", AppRegisterDecision{Authorizers: []*auth.AuthorizerAssignment{valid}}, true},
		{"nil entry", AppRegisterDecision{Authorizers: []*auth.AuthorizerAssignment{nil}}, false},
		{"names an actor", AppRegisterDecision{Authorizers: []*auth.AuthorizerAssignment{{Action: "a", Endpoint: "astral://x:y", Actor: actor}}}, false},
		{"no endpoint", AppRegisterDecision{Authorizers: []*auth.AuthorizerAssignment{{Action: "a"}}}, false},
	} {
		if err := tc.d.Validate(); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v", tc.name, err)
		}
	}
}
