package apphost

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// grantArgs mirrors astrald's opGrantArgs (mod/apphost/src/op_grant.go), so
// the sent query binds the way the node binds it: a misnamed key is skipped
// and leaves its field zero.
type grantArgs struct {
	Identity    string
	Action      astral.String8
	Duration    astral.Duration
	Constrained bool
}

func boundGrantArgs(t *testing.T, router *recordingRouter) grantArgs {
	t.Helper()

	path, params := query.Parse(router.queryString)
	if path != apphost.MethodGrant {
		t.Fatalf("path: want %v, got %q", apphost.MethodGrant, path)
	}

	var args grantArgs
	if err := query.Edit(&args).SetMany(params); err != nil {
		t.Fatalf("bind %v: %v", params, err)
	}
	return args
}

// An unconstrained grant sends identity and action alone and writes nothing
// on the channel.
func TestGrant_SendsIdentityAndAction(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, &astral.Ack{})

	err := client.Grant(astral.NewContext(nil), app, "mod.auth.see_node_state_action", GrantOptions{})
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}

	args := boundGrantArgs(t, router)
	if args.Identity != app.String() || args.Action != "mod.auth.see_node_state_action" {
		t.Fatalf("args: got %+v", args)
	}
	if args.Duration != 0 || args.Constrained {
		t.Fatalf("an unconstrained grant until revoked sent %+v", args)
	}
	if router.sent.Len() != 0 {
		t.Fatalf("an unconstrained grant wrote %d bytes on the channel", router.sent.Len())
	}
}

// A constrained grant sets constrained and writes the constraints as one
// bundle, the way the services topic grants discovery to an app.
func TestGrant_SendsTheConstraintsBundle(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, &astral.Ack{})

	scope := astral.NewBundle()
	if err := scope.Append(&services.DiscoveryScope{Rules: []*services.DiscoveryRule{{Services: []astral.String8{"inference"}}}}); err != nil {
		t.Fatal(err)
	}

	err := client.Grant(astral.NewContext(nil), app, services.ServiceDiscoveryAction{}.ObjectType(), GrantOptions{
		Duration:    astral.Duration(720 * time.Hour),
		Constraints: scope,
	})
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}

	args := boundGrantArgs(t, router)
	if args.Identity != app.String() || string(args.Action) != "mod.services.service_discovery_action" {
		t.Fatalf("args: got %+v", args)
	}
	if !args.Constrained || args.Duration.Duration() != 720*time.Hour {
		t.Fatalf("args: want constrained for 720h, got %+v", args)
	}

	sent := decodeAll(t, router.sent.Bytes())
	if len(sent) != 1 {
		t.Fatalf("sent %d objects; want one bundle", len(sent))
	}
	bundle, ok := sent[0].(*astral.Bundle)
	if !ok {
		t.Fatalf("sent %s; want a bundle", sent[0].ObjectType())
	}
	got, found := astral.First[*services.DiscoveryScope](bundle.Objects())
	if !found || len(got.Rules) != 1 || len(got.Rules[0].Services) != 1 || got.Rules[0].Services[0] != "inference" {
		t.Fatalf("constraints: got %v", bundle)
	}
}

func TestGrant_ErrorMessageIsReturned(t *testing.T) {
	client, _ := recordingClient(t, astral.NewError("missing identity"))

	err := client.Grant(astral.NewContext(nil), astral.GenerateIdentity(), "mod.auth.see_node_state_action", GrantOptions{})
	if err == nil || err.Error() != "missing identity" {
		t.Fatalf("want error %q, got %v", "missing identity", err)
	}
}

// A stream that ends without an ack recorded nothing, so it is not a grant.
func TestGrant_NoAckIsAnError(t *testing.T) {
	client, _ := recordingClient(t)

	err := client.Grant(astral.NewContext(nil), astral.GenerateIdentity(), "mod.auth.see_node_state_action", GrantOptions{})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
	}
}
