package sys

import (
	"bytes"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// TestRegisterManagerRequestRoundTrip requires a request to survive a binary
// and a JSON round trip unchanged.
func TestRegisterManagerRequestRoundTrip(t *testing.T) {
	obj := &RegisterManagerRequest{
		Name:         "control-plane",
		GrantPermits: []*auth.Permit{{Action: "mod.user.see_swarm_action"}},
		Policies: []*PolicyBinding{
			{Policy: PolicyAppRegister, Operation: "cp.approve_app"},
			{Policy: PolicySwarmJoin, Operation: "cp.approve_join", Timeout: astral.Duration(time.Minute)},
		},
	}

	var first bytes.Buffer
	if _, err := obj.WriteTo(&first); err != nil {
		t.Fatalf("write: %v", err)
	}
	want := append([]byte(nil), first.Bytes()...)
	var fresh RegisterManagerRequest
	if _, err := fresh.ReadFrom(&first); err != nil {
		t.Fatalf("read: %v", err)
	}
	var second bytes.Buffer
	if _, err := fresh.WriteTo(&second); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if !bytes.Equal(want, second.Bytes()) {
		t.Fatal("binary round trip changed the request")
	}

	j, err := obj.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	var fromJSON RegisterManagerRequest
	if err := fromJSON.UnmarshalJSON(j); err != nil {
		t.Fatalf("unmarshal json: %v (%s)", err, j)
	}
	j2, err := fromJSON.MarshalJSON()
	if err != nil {
		t.Fatalf("remarshal json: %v", err)
	}
	if !bytes.Equal(j, j2) {
		t.Fatalf("json round trip changed the request: %s != %s", j, j2)
	}
}

func TestSysObjectTypes(t *testing.T) {
	for want, obj := range map[string]astral.Object{
		"mod.sys.register_manager_request": &RegisterManagerRequest{},
		"mod.sys.policy_binding":           &PolicyBinding{},
	} {
		if got := obj.ObjectType(); got != want {
			t.Errorf("%T.ObjectType() = %q, want %q", obj, got, want)
		}
		if _, ok := astral.New(want).(astral.Object); !ok {
			t.Errorf("%q is not registered", want)
		}
	}
}
