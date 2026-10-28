package objects

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

type localRouter struct {
	inner astral.Router
	id    *astral.Identity
}

func (r *localRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (r *localRouter) GuestID() *astral.Identity { return r.id }
func (r *localRouter) HostID() *astral.Identity  { return r.id }

// clientFor routes the real client to op standing in for the node's method,
// so the wire argument names are pinned.
func clientFor(t *testing.T, method string, op any) *Client {
	t.Helper()
	router := routing.NewOpRouter()
	o, err := routing.NewOp(op)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.AddOp(method, o); err != nil {
		t.Fatal(err)
	}
	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: router, id: id}))
}

// getBlueprintArgs declares the arguments as astrald's op does.
type getBlueprintArgs struct {
	Type string `query:"required"`
	Out  string
}

func TestGetBlueprint(t *testing.T) {
	want := astral.NewBlueprint("test.get_blueprint",
		astral.Field{Name: "N", Spec: &astral.PrimitiveSpec{PrimitiveType: "uint32"}})
	c := clientFor(t, objects.MethodGetBlueprint, func(_ *astral.Context, q *routing.IncomingQuery, args getBlueprintArgs) error {
		ch := q.Accept()
		defer ch.Close()
		if args.Type != want.Type.String() {
			return ch.Send(astral.NewError("blueprint not found: " + args.Type))
		}
		return ch.Send(want)
	})
	ctx := astral.NewContext(nil)

	got, err := c.GetBlueprint(ctx, "test.get_blueprint")
	if err != nil {
		t.Fatal(err)
	}
	gotID, _ := astral.ResolveObjectID(got)
	wantID, _ := astral.ResolveObjectID(want)
	if !gotID.IsEqual(wantID) {
		t.Fatalf("got %+v; want %+v", got, want)
	}

	_, err = c.GetBlueprint(ctx, "test.unknown")
	if err == nil || err.Error() != "blueprint not found: test.unknown" {
		t.Fatalf("unknown type: err = %v", err)
	}
}

// A node that closes the channel without an answer is an error, not a nil
// blueprint.
func TestGetBlueprintEmptyAnswer(t *testing.T) {
	c := clientFor(t, objects.MethodGetBlueprint, func(_ *astral.Context, q *routing.IncomingQuery, _ getBlueprintArgs) error {
		return q.Accept().Close()
	})

	bp, err := c.GetBlueprint(astral.NewContext(nil), "test.get_blueprint")
	if !errors.Is(err, io.ErrUnexpectedEOF) || bp != nil {
		t.Fatalf("got %v, %v; want io.ErrUnexpectedEOF", bp, err)
	}
}
