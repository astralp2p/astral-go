package apphost

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
)

// roundTrip encodes obj, decodes it into fresh, and requires the second
// encoding to equal the first, in binary and in JSON.
func roundTrip(t *testing.T, obj, fresh, freshJSON astral.Object) {
	t.Helper()

	var first bytes.Buffer
	if _, err := obj.WriteTo(&first); err != nil {
		t.Fatalf("write: %v", err)
	}
	want := append([]byte(nil), first.Bytes()...)
	if _, err := fresh.ReadFrom(&first); err != nil {
		t.Fatalf("read: %v", err)
	}
	var second bytes.Buffer
	if _, err := fresh.WriteTo(&second); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if !bytes.Equal(want, second.Bytes()) {
		t.Fatalf("binary round trip changed %s", obj.ObjectType())
	}

	j, err := astral.Objectify(obj).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if err := astral.Objectify(freshJSON).UnmarshalJSON(j); err != nil {
		t.Fatalf("unmarshal json: %v (%s)", err, j)
	}
	j2, err := astral.Objectify(freshJSON).MarshalJSON()
	if err != nil {
		t.Fatalf("remarshal json: %v", err)
	}
	if !bytes.Equal(j, j2) {
		t.Fatalf("json round trip changed %s:\n%s\n%s", obj.ObjectType(), j, j2)
	}
}

func bundle(t *testing.T, objects ...astral.Object) *astral.Bundle {
	t.Helper()
	b := astral.NewBundle()
	if err := b.Append(objects...); err != nil {
		t.Fatal(err)
	}
	return b
}

func discoveryPermit(t *testing.T, service astral.String8) *auth.Permit {
	t.Helper()
	return &auth.Permit{
		Action: astral.String8(services.ServiceDiscoveryAction{}.ObjectType()),
		Constraints: bundle(t, &services.DiscoveryScope{Rules: []*services.DiscoveryRule{
			{Services: []astral.String8{service}},
		}}),
	}
}

func expiry() *astral.Time {
	at := astral.Time(time.Date(2027, 1, 2, 3, 4, 5, 6, time.UTC))
	return &at
}

func TestApphostWireTypesRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name             string
		obj, fresh, json astral.Object
	}{
		{
			name:  "grant with scope and expiry",
			obj:   &Grant{ID: 7, Permit: discoveryPermit(t, "inference"), ExpiresAt: expiry()},
			fresh: &Grant{}, json: &Grant{},
		},
		{
			name:  "grant until revoked",
			obj:   &Grant{ID: 8, Permit: &auth.Permit{Action: "mod.auth.see_node_state_action"}},
			fresh: &Grant{}, json: &Grant{},
		},
		{
			name: "register request",
			obj: &RegisterRequest{
				Name:            "smithd",
				GrantPermits:    []*auth.Permit{discoveryPermit(t, "inference")},
				ContractPermits: []*auth.Permit{{Action: "mod.user.see_swarm_action"}},
			},
			fresh: &RegisterRequest{}, json: &RegisterRequest{},
		},
		{
			name: "grant request",
			obj: &GrantRequest{
				Identity: astral.GenerateIdentity(),
				Duration: astral.Duration(720 * time.Hour),
				Permits:  []*auth.Permit{discoveryPermit(t, "inference"), {Action: "mod.auth.see_objects_action"}},
			},
			fresh: &GrantRequest{}, json: &GrantRequest{},
		},
		{
			name:  "request grant request until revoked",
			obj:   &RequestGrantRequest{Permits: []*auth.Permit{discoveryPermit(t, "inference")}},
			fresh: &RequestGrantRequest{}, json: &RequestGrantRequest{},
		},
		{
			name: "request grant request with duration",
			obj: &RequestGrantRequest{
				Permits:  []*auth.Permit{discoveryPermit(t, "inference")},
				Duration: astral.Duration(time.Hour),
			},
			fresh: &RequestGrantRequest{}, json: &RequestGrantRequest{},
		},
		{
			name: "grant policy request",
			obj: &GrantPolicyRequest{
				Requester: astral.GenerateIdentity(),
				Origin:    "https://app.example",
				Permits:   []*auth.Permit{discoveryPermit(t, "inference")},
				Duration:  astral.Duration(time.Hour),
			},
			fresh: &GrantPolicyRequest{}, json: &GrantPolicyRequest{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roundTrip(t, tc.obj, tc.fresh, tc.json)
		})
	}
}

// decodeGrant decodes in through the binary codec and through the JSON codec.
func decodeGrant(t *testing.T, in *Grant) (bin, js *Grant) {
	t.Helper()

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	bin = &Grant{}
	if _, err := bin.ReadFrom(&buf); err != nil {
		t.Fatalf("read: %v", err)
	}

	j, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	js = &Grant{}
	if err := json.Unmarshal(j, js); err != nil {
		t.Fatalf("unmarshal json: %v (%s)", err, j)
	}
	return bin, js
}

func TestGrantFieldsSurviveDecode(t *testing.T) {
	set := &Grant{ID: 7, Permit: discoveryPermit(t, "inference"), ExpiresAt: expiry()}
	until := &Grant{ID: 9, Permit: &auth.Permit{Action: "mod.auth.see_node_state_action"}}

	bin, js := decodeGrant(t, set)
	for codec, got := range map[string]*Grant{"binary": bin, "json": js} {
		if got.ID != 7 || got.Permit == nil || got.Permit.Action != set.Permit.Action {
			t.Fatalf("%s: got %+v", codec, got)
		}
		scope, ok := astral.First[*services.DiscoveryScope](got.Permit.Constraints.Objects())
		if !ok || len(scope.Rules) != 1 || len(scope.Rules[0].Services) != 1 || scope.Rules[0].Services[0] != "inference" {
			t.Fatalf("%s: constraints arrived as %+v", codec, got.Permit.Constraints.Objects())
		}
		if got.ExpiresAt == nil || !got.ExpiresAt.Time().Equal(set.ExpiresAt.Time()) {
			t.Fatalf("%s: ExpiresAt arrived as %v, want %v", codec, got.ExpiresAt, set.ExpiresAt)
		}
	}

	bin, js = decodeGrant(t, until)
	for codec, got := range map[string]*Grant{"binary": bin, "json": js} {
		if got.ID != 9 || got.Permit == nil || got.Permit.Action != until.Permit.Action {
			t.Fatalf("%s: got %+v", codec, got)
		}
		if got.ExpiresAt != nil {
			t.Fatalf("%s: until-revoked grant arrived with ExpiresAt %v", codec, got.ExpiresAt)
		}
	}
}

// The spec writes an until-revoked ExpiresAt as JSON null.
func TestGrantUntilRevokedIsJSONNull(t *testing.T) {
	j, err := json.Marshal(&Grant{ID: 1, Permit: &auth.Permit{Action: "mod.auth.see_node_state_action"}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(j, []byte(`"ExpiresAt":null`)) {
		t.Fatalf("until-revoked grant marshals as %s", j)
	}
}

func TestApphostObjectTypes(t *testing.T) {
	for want, obj := range map[string]astral.Object{
		"mod.apphost.grant":                 &Grant{},
		"mod.apphost.register_request":      &RegisterRequest{},
		"mod.apphost.grant_request":         &GrantRequest{},
		"mod.apphost.request_grant_request": &RequestGrantRequest{},
		"mod.apphost.grant_policy_request":  &GrantPolicyRequest{},
		"mod.apphost.grant_decision":        &GrantDecision{},
	} {
		if got := obj.ObjectType(); got != want {
			t.Errorf("%T.ObjectType() = %q, want %q", obj, got, want)
		}
	}

	if _, ok := astral.New("mod.apphost.grant_request").(*GrantRequest); !ok {
		t.Errorf("mod.apphost.grant_request does not decode as GrantRequest")
	}
	if _, ok := astral.New("mod.apphost.grant_policy_request").(*GrantPolicyRequest); !ok {
		t.Errorf("mod.apphost.grant_policy_request does not decode as GrantPolicyRequest")
	}
}
