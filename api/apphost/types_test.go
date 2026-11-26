package apphost

import (
	"bytes"
	"encoding/json"
	"testing"

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

func TestApphostWireTypesRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name             string
		obj, fresh, json astral.Object
	}{
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
			name: "register request with both switches off",
			obj: &RegisterRequest{
				Name:      "smithd",
				Vfs:       astral.NewBool(false),
				Evaluator: astral.NewBool(false),
			},
			fresh: &RegisterRequest{}, json: &RegisterRequest{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roundTrip(t, tc.obj, tc.fresh, tc.json)
		})
	}
}

// decodeRegisterRequest decodes in through the binary codec and through the JSON codec.
func decodeRegisterRequest(t *testing.T, in *RegisterRequest) map[string]*RegisterRequest {
	t.Helper()

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	bin := &RegisterRequest{}
	if _, err := bin.ReadFrom(&buf); err != nil {
		t.Fatalf("read: %v", err)
	}

	j, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	js := &RegisterRequest{}
	if err := json.Unmarshal(j, js); err != nil {
		t.Fatalf("unmarshal json: %v (%s)", err, j)
	}

	return map[string]*RegisterRequest{"binary": bin, "json": js}
}

// TestRegisterRequestSwitchesSurviveDecode: each value of Vfs and Evaluator, nil included, arrives as sent and
// reads as the request it is.
func TestRegisterRequestSwitchesSurviveDecode(t *testing.T) {
	on, off := astral.NewBool(true), astral.NewBool(false)

	for _, tc := range []struct {
		name           string
		vfs, evaluator *astral.Bool
		wantVFS        bool
		wantEvaluator  bool
	}{
		{"omitted", nil, nil, true, true},
		{"both on", on, on, true, true},
		{"evaluator off", nil, off, true, false},
		{"vfs off", off, nil, false, false},
		{"vfs off evaluator on", off, on, false, false},
	} {
		in := &RegisterRequest{Name: "app", Vfs: tc.vfs, Evaluator: tc.evaluator}
		for codec, got := range decodeRegisterRequest(t, in) {
			if (got.Vfs == nil) != (tc.vfs == nil) || (got.Evaluator == nil) != (tc.evaluator == nil) {
				t.Errorf("%s/%s: nil switches changed: %+v", tc.name, codec, got)
			}
			if got.RequestsVFS() != tc.wantVFS || got.RequestsEvaluator() != tc.wantEvaluator {
				t.Errorf("%s/%s: requests VFS %v, evaluator %v; want %v, %v", tc.name, codec,
					got.RequestsVFS(), got.RequestsEvaluator(), tc.wantVFS, tc.wantEvaluator)
			}
		}
	}
}

// TestRegisterRequestJSONWithoutSwitchesAsksForBoth: a JSON request written before the switches existed asks for
// a VFS and the evaluator.
func TestRegisterRequestJSONWithoutSwitchesAsksForBoth(t *testing.T) {
	var req RegisterRequest
	if err := json.Unmarshal([]byte(`{"Name":"app","GrantPermits":[],"ContractPermits":[]}`), &req); err != nil {
		t.Fatal(err)
	}
	if !req.RequestsVFS() || !req.RequestsEvaluator() {
		t.Fatalf("a request without switches asks for VFS %v, evaluator %v", req.RequestsVFS(), req.RequestsEvaluator())
	}
}

func TestApphostObjectTypes(t *testing.T) {
	for want, obj := range map[string]astral.Object{
		"mod.apphost.register_request": &RegisterRequest{},
	} {
		if got := obj.ObjectType(); got != want {
			t.Errorf("%T.ObjectType() = %q, want %q", obj, got, want)
		}
	}
}
