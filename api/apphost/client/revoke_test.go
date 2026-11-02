package apphost

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// revokeParams requires the routed query to be apphost.revoke and returns its
// arguments.
func revokeParams(t *testing.T, router *recordingRouter) map[string]string {
	t.Helper()
	path, params := query.Parse(router.queryString)
	if path != "apphost.revoke" {
		t.Fatalf("path: got %q", path)
	}
	return params
}

func TestRevokeGrant_SendsIdentityAndID(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, &astral.Ack{})

	if err := client.RevokeGrant(astral.NewContext(nil), app, 42); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}

	params := revokeParams(t, router)
	if len(params) != 2 || params["identity"] != app.String() || params["id"] != "42" {
		t.Fatalf("params: got %v", params)
	}
}

func TestRevokeAction_SendsIdentityAndAction(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, &astral.Ack{})

	if err := client.RevokeAction(astral.NewContext(nil), app, "mod.services.service_discovery_action"); err != nil {
		t.Fatalf("RevokeAction: %v", err)
	}

	params := revokeParams(t, router)
	if len(params) != 2 || params["identity"] != app.String() || params["action"] != "mod.services.service_discovery_action" {
		t.Fatalf("params: got %v", params)
	}
}

func TestRevoke_ErrorMessageIsReturned(t *testing.T) {
	for name, revoke := range map[string]func(*Client) error{
		"grant": func(c *Client) error { return c.RevokeGrant(astral.NewContext(nil), astral.GenerateIdentity(), 42) },
		"action": func(c *Client) error {
			return c.RevokeAction(astral.NewContext(nil), astral.GenerateIdentity(), "mod.auth.see_objects_action")
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := recordingClient(t, astral.NewError("grant not found"))
			if err := revoke(client); err == nil || err.Error() != "grant not found" {
				t.Fatalf("want error %q, got %v", "grant not found", err)
			}
		})
	}
}

func TestRevoke_NoAckIsAnError(t *testing.T) {
	for name, revoke := range map[string]func(*Client) error{
		"grant": func(c *Client) error { return c.RevokeGrant(astral.NewContext(nil), astral.GenerateIdentity(), 42) },
		"action": func(c *Client) error {
			return c.RevokeAction(astral.NewContext(nil), astral.GenerateIdentity(), "mod.auth.see_objects_action")
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := recordingClient(t)
			if err := revoke(client); !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
			}
		})
	}
}
