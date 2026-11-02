package apphost

import (
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// ListGrants sends exactly the identity argument apphost.list_grants reads.
func TestListGrants_SendsTheIdentity(t *testing.T) {
	app := astral.GenerateIdentity()
	client, router := recordingClient(t, &astral.EOS{})

	if _, err := client.ListGrants(astral.NewContext(nil), app); err != nil {
		t.Fatalf("ListGrants: %v", err)
	}

	path, params := query.Parse(router.queryString)
	if path != "apphost.list_grants" {
		t.Fatalf("path: got %q", path)
	}
	if len(params) != 1 || params["identity"] != app.String() {
		t.Fatalf("params: got %v", params)
	}
}

func TestListGrants_ReturnsGrantsUntilEOS(t *testing.T) {
	at := astral.Time(time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC))
	client, _ := recordingClient(t,
		&apphost.Grant{ID: 7, Permit: discoveryPermit(t, "inference")},
		&apphost.Grant{ID: 9, Permit: &auth.Permit{Action: "mod.auth.see_objects_action"}, ExpiresAt: &at},
		&astral.EOS{},
	)

	grants, err := client.ListGrants(astral.NewContext(nil), astral.GenerateIdentity())
	if err != nil {
		t.Fatalf("ListGrants: %v", err)
	}
	if len(grants) != 2 || grants[0].ID != 7 || grants[1].ID != 9 {
		t.Fatalf("grants: got %+v", grants)
	}
	if grants[0].ExpiresAt != nil {
		t.Fatalf("an until-revoked grant arrived with ExpiresAt %v", grants[0].ExpiresAt)
	}
	if grants[1].ExpiresAt == nil || !grants[1].ExpiresAt.Time().Equal(at.Time()) {
		t.Fatalf("ExpiresAt: want %v, got %v", at, grants[1].ExpiresAt)
	}
}

func TestListGrants_ErrorMessageIsReturned(t *testing.T) {
	client, _ := recordingClient(t, astral.NewError("not permitted"))

	_, err := client.ListGrants(astral.NewContext(nil), astral.GenerateIdentity())
	if err == nil || err.Error() != "not permitted" {
		t.Fatalf("want error %q, got %v", "not permitted", err)
	}
}

// A listing cut short before EOS is not a complete list.
func TestListGrants_NoEOSIsAnError(t *testing.T) {
	client, _ := recordingClient(t, &apphost.Grant{ID: 7, Permit: &auth.Permit{Action: "mod.auth.see_objects_action"}})

	grants, err := client.ListGrants(astral.NewContext(nil), astral.GenerateIdentity())
	if err == nil {
		t.Fatalf("a listing without EOS returned %+v and no error", grants)
	}
}
