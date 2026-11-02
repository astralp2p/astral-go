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

func TestRequestGrant_SendsOneRequestGrantRequestAndNoArguments(t *testing.T) {
	client, router := recordingClient(t, &astral.Ack{})

	err := client.RequestGrant(astral.NewContext(nil), &apphost.RequestGrantRequest{
		Permits:  []*auth.Permit{discoveryPermit(t, "inference")},
		Duration: astral.Duration(time.Hour),
	})
	if err != nil {
		t.Fatalf("RequestGrant: %v", err)
	}

	noArguments(t, router, "apphost.request_grant")
	req := onlyObject[*apphost.RequestGrantRequest](t, router)
	if req.Duration.Duration() != time.Hour {
		t.Fatalf("duration: got %v", req.Duration)
	}
	if len(req.Permits) != 1 || req.Permits[0].Action != "mod.services.service_discovery_action" {
		t.Fatalf("permits arrived as %+v", req.Permits)
	}
	if got := scopeServices(t, req.Permits[0]); len(got) != 1 || got[0] != "inference" {
		t.Fatalf("scope services: got %v", got)
	}
}

func TestRequestGrant_DeclineMatchesSentinel(t *testing.T) {
	client, _ := recordingClient(t, apphost.ErrGrantDeclined)

	err := client.RequestGrant(astral.NewContext(nil), &apphost.RequestGrantRequest{Permits: []*auth.Permit{discoveryPermit(t, "inference")}})
	if !errors.Is(err, apphost.ErrGrantDeclined) {
		t.Fatalf("want apphost.ErrGrantDeclined, got %v", err)
	}
}

func TestRequestGrant_InvalidPermitRefusalMatchesSentinel(t *testing.T) {
	client, _ := recordingClient(t, astral.NewError("invalid permit: mod.services.service_discovery_action: requires constraints"))

	err := client.RequestGrant(astral.NewContext(nil), &apphost.RequestGrantRequest{Permits: []*auth.Permit{{Action: "mod.services.service_discovery_action"}}})
	if !errors.Is(err, auth.ErrInvalidPermit) {
		t.Fatalf("want auth.ErrInvalidPermit, got %v", err)
	}
}

func TestRequestGrant_NoAckIsAnError(t *testing.T) {
	client, _ := recordingClient(t)

	err := client.RequestGrant(astral.NewContext(nil), &apphost.RequestGrantRequest{})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("want io.ErrUnexpectedEOF, got %v", err)
	}
}

func TestRequestGrant_NilRequestIsRefused(t *testing.T) {
	client, router := recordingClient(t, &astral.Ack{})

	err := client.RequestGrant(astral.NewContext(nil), nil)
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
	if router.queryString != "" {
		t.Fatalf("a nil request routed %q", router.queryString)
	}
}
