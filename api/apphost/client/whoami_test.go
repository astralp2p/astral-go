package apphost

import (
	"errors"
	"testing"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

func TestWhoami_ReturnsTheSessionIdentity(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, app)

	id, err := client.Whoami(astral.NewContext(nil))
	if err != nil {
		t.Fatalf("Whoami: %v", err)
	}
	if !id.IsEqual(app) {
		t.Fatalf("identity: want %v, got %v", app, id)
	}

	path, params := query.Parse(router.queryString)
	if path != apphost.MethodWhoami || len(params) != 0 {
		t.Fatalf("query: want %v with no arguments, got %q", apphost.MethodWhoami, router.queryString)
	}
}

// refusingRouter fails every query the way a session fails whose token the
// node did not accept.
type refusingRouter struct{ err error }

func (r refusingRouter) RouteQuery(*astral.Context, *astral.InFlightQuery) (astral.Conn, error) {
	return nil, r.err
}

func (refusingRouter) GuestID() *astral.Identity { return nil }
func (refusingRouter) HostID() *astral.Identity  { return nil }

// A rejected token stays recognisable, so a caller can tell it from a node
// it could not reach.
func TestWhoami_RejectedTokenIsReturned(t *testing.T) {
	client := New(nil, astrald.New(refusingRouter{&apphost.ErrorMsg{Code: apphost.ErrCodeAuthFailed}}))

	id, err := client.Whoami(astral.NewContext(nil))
	var msg *apphost.ErrorMsg
	if !errors.As(err, &msg) || msg.Code != apphost.ErrCodeAuthFailed {
		t.Fatalf("want the %s error message, got %v", apphost.ErrCodeAuthFailed, err)
	}
	if id != nil {
		t.Fatalf("want no identity, got %v", id)
	}
}

func TestWhoami_ErrorMessageIsReturned(t *testing.T) {
	client, _ := recordingClient(t, astral.NewError("internal error"))

	if _, err := client.Whoami(astral.NewContext(nil)); err == nil || err.Error() != "internal error" {
		t.Fatalf("want error %q, got %v", "internal error", err)
	}
}

// An answer without an identity names nobody, so it must not read as one.
func TestWhoami_EmptyAnswerIsAProtocolError(t *testing.T) {
	client, _ := recordingClient(t)

	id, err := client.Whoami(astral.NewContext(nil))
	if !errors.Is(err, apphost.ErrProtocolError) {
		t.Fatalf("want ErrProtocolError, got %v", err)
	}
	if id != nil {
		t.Fatalf("want no identity, got %v", id)
	}
}
