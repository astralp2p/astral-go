package apphost

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

func seeNodeState() *apphost.GrantRequest {
	return &apphost.GrantRequest{
		Identity: astral.GenerateIdentity(),
		Permits:  []*auth.Permit{{Action: "mod.auth.see_node_state_action"}},
	}
}

// A grant sends no query arguments and one GrantRequest carrying the identity,
// the duration and each permit with its constraints, the way the services
// topic grants discovery to an app.
func TestGrant_SendsOneGrantRequestAndNoArguments(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, &astral.Ack{})

	err := client.Grant(astral.NewContext(nil), &apphost.GrantRequest{
		Identity: app,
		Duration: astral.Duration(720 * time.Hour),
		Permits:  []*auth.Permit{discoveryPermit(t, "inference")},
	})
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}

	noArguments(t, router, "apphost.grant")
	req := onlyObject[*apphost.GrantRequest](t, router)
	if !req.Identity.IsEqual(app) {
		t.Fatalf("identity: want %v, got %v", app, req.Identity)
	}
	if req.Duration.Duration() != 720*time.Hour {
		t.Fatalf("duration: got %v", req.Duration)
	}
	if len(req.Permits) != 1 || req.Permits[0].Action != "mod.services.service_discovery_action" {
		t.Fatalf("permits arrived as %+v", req.Permits)
	}
	if got := scopeServices(t, req.Permits[0]); len(got) != 1 || got[0] != "inference" {
		t.Fatalf("scope services: got %v", got)
	}
}

func TestGrant_ErrorMessageIsReturned(t *testing.T) {
	client, _ := recordingClient(t, astral.NewError("missing identity"))

	err := client.Grant(astral.NewContext(nil), seeNodeState())
	if err == nil || err.Error() != "missing identity" {
		t.Fatalf("want error %q, got %v", "missing identity", err)
	}
}

// A stream that ends without an ack recorded nothing, so it is not a grant.
func TestGrant_NoAckIsAnError(t *testing.T) {
	client, _ := recordingClient(t)

	err := client.Grant(astral.NewContext(nil), seeNodeState())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
	}
}

// The node's invalid-permit refusal crosses the wire as text, and still
// matches the sentinel.
func TestGrant_InvalidPermitRefusalMatchesSentinel(t *testing.T) {
	client, _ := recordingClient(t, astral.NewError("invalid permit: mod.services.service_discovery_action: requires constraints"))

	err := client.Grant(astral.NewContext(nil), seeNodeState())
	if !errors.Is(err, auth.ErrInvalidPermit) {
		t.Fatalf("want auth.ErrInvalidPermit, got %v", err)
	}
}

func TestGrant_NilRequestIsRefused(t *testing.T) {
	client, router := recordingClient(t, &astral.Ack{})

	err := client.Grant(astral.NewContext(nil), nil)
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
	if router.queryString != "" {
		t.Fatalf("a nil request routed %q", router.queryString)
	}
}
