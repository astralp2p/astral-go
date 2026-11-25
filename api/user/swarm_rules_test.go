package user

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// An entry's ID is the ID of its signed form, stable across a round trip and
// different from the body's ID, so a reference made from the stored entry
// finds it again.
func TestEntryID_IsTheSignedForm(t *testing.T) {
	for _, e := range signedEntries() {
		id, err := astral.ResolveObjectID(e)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if _, err := astral.Encode(&buf, e); err != nil {
			t.Fatalf("%s: encode: %v", e.ObjectType(), err)
		}
		got, _, err := astral.Decode(&buf)
		if err != nil {
			t.Fatalf("%s: decode: %v", e.ObjectType(), err)
		}
		gotID, err := astral.ResolveObjectID(got)
		if err != nil || *gotID != *id {
			t.Errorf("%s: entry ID changed across a round trip", e.ObjectType())
		}

		var body astral.Object
		switch s := e.(type) {
		case *SignedContractRequest:
			body = s.ContractRequest
		case *SignedJoinRequest:
			body = s.JoinRequest
		case *SignedExpelRequest:
			body = s.ExpelRequest
		case *SignedApproval:
			body = s.Approval
		case *SignedDecline:
			body = s.Decline
		}
		bodyID, err := astral.ResolveObjectID(body)
		if err != nil || *bodyID == *id {
			t.Errorf("%s: entry ID must differ from the body's ID", e.ObjectType())
		}
	}
}

func TestMayAnswer(t *testing.T) {
	a, b := astral.GenerateIdentity(), astral.GenerateIdentity()
	cr := NewContractRequest(a, b, nil)
	er := NewExpelRequest(a, b)
	join := func(t Tier) *JoinRequest { return NewJoinRequest(a, b, t) }

	for _, tc := range []struct {
		name string
		tier Tier
		req  astral.Object
		want bool
	}{
		{"privileged contract", TierPrivilegedMember, cr, true},
		{"privileged signed contract", TierPrivilegedMember, &SignedContractRequest{ContractRequest: cr}, true},
		{"privileged expel", TierPrivilegedMember, er, true},
		{"privileged privileged join", TierPrivilegedMember, join(TierPrivilegedMember), true},
		{"trusted member join", TierTrusted, join(TierMember), true},
		{"trusted signed member join", TierTrusted, &SignedJoinRequest{JoinRequest: join(TierMember)}, true},
		{"trusted trusted join", TierTrusted, join(TierTrusted), false},
		{"trusted contract", TierTrusted, cr, false},
		{"trusted expel", TierTrusted, er, false},
		{"member member join", TierMember, join(TierMember), false},
		{"member contract", TierMember, cr, false},
		{"invalid tier", Tier("root"), cr, false},
		{"not a request", TierPrivilegedMember, NewApproval(a, sampleRequestID()), false},
		{"nil signed body", TierPrivilegedMember, &SignedJoinRequest{}, false},
	} {
		if got := MayAnswer(tc.tier, tc.req); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestContractRequest_Covers(t *testing.T) {
	a, b := astral.GenerateIdentity(), astral.GenerateIdentity()
	p1 := &auth.Permit{Action: "test.one", Delegation: 1}
	p2 := &auth.Permit{Action: "test.two"}
	req := NewContractRequest(a, b, []*auth.Permit{p1, p2})
	contract := func(ps ...*auth.Permit) *auth.Contract { return &auth.Contract{Issuer: a, Subject: b, Permits: ps} }

	for _, tc := range []struct {
		name string
		c    *auth.Contract
		want bool
	}{
		{"exact copy", contract(p1, p2), true},
		{"reordered subset", contract(p2), true},
		{"no permits", contract(), false},
		{"more delegation", contract(&auth.Permit{Action: "test.one", Delegation: 2}), false},
		{"other action", contract(&auth.Permit{Action: "test.three"}), false},
		{"added constraint", contract(&auth.Permit{Action: "test.two", Constraints: constrained(astral.NewString8("x"))}), false},
		{"nil contract", nil, false},
	} {
		if got := req.Covers(tc.c); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestExpelAction(t *testing.T) {
	sent := &ExpelAction{Action: auth.NewAction(astral.GenerateIdentity()), Subject: astral.GenerateIdentity()}
	var buf bytes.Buffer
	if _, err := sent.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	var got ExpelAction
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatal(err)
	}
	if !got.Subject.IsEqual(sent.Subject) || got.Nonce != sent.Nonce {
		t.Error("ExpelAction changed across a round trip")
	}
	if !got.ApplyConstraints(nil) || got.ApplyConstraints(constrained(astral.NewString8("x"))) {
		t.Error("ExpelAction refuses constraints")
	}
}
