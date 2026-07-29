package query

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// The route helpers are the caller and responder halves of a query: RouteInFlight
// drives a query through a Router and hands back a Conn; Accept, Reject, and
// RouteNotFound are the three answers a responder can give.

// routerFunc adapts a function to astral.Router.
type routerFunc func(ctx *astral.Context, q *astral.InFlightQuery, w io.WriteCloser) (io.WriteCloser, error)

func (f routerFunc) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery, w io.WriteCloser) (io.WriteCloser, error) {
	return f(ctx, q, w)
}

// nopWriteCloser is a discarding sink the responder can hand back.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func testQuery(t *testing.T) *astral.Query {
	t.Helper()

	id := astral.GenerateIdentity()
	return New(id, id, "op", nil)
}

// On success the caller gets an outbound Conn wired to the identities of the
// query.
func TestRouteInFlight_Success(t *testing.T) {
	ctx := astral.NewContext(nil)
	q := testQuery(t)

	var seen *astral.InFlightQuery
	router := routerFunc(func(_ *astral.Context, q *astral.InFlightQuery, _ io.WriteCloser) (io.WriteCloser, error) {
		seen = q
		return nopWriteCloser{Writer: io.Discard}, nil
	})

	conn, err := RouteInFlight(ctx, router, astral.Launch(q))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("want a connection, got nil")
	}

	if seen == nil || seen.QueryString != "op" {
		t.Fatalf("want the router to see the query, got %v", seen)
	}
	if !conn.(*Conn).Outbound() {
		t.Fatal("Outbound: want true for a routed query")
	}
	if !conn.LocalIdentity().IsEqual(q.Caller) {
		t.Fatal("LocalIdentity: want the caller")
	}
	if !conn.RemoteIdentity().IsEqual(q.Target) {
		t.Fatal("RemoteIdentity: want the target")
	}
}

// A router error reaches the caller unwrapped, and no connection is returned.
func TestRouteInFlight_PropagatesRouterErrors(t *testing.T) {
	ctx := astral.NewContext(nil)

	cases := []struct {
		name string
		err  error
	}{
		{"rejected", astral.NewErrRejected(7)},
		{"route not found", astral.NewErrRouteNotFound()},
		{"arbitrary", errors.New("boom")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := routerFunc(func(*astral.Context, *astral.InFlightQuery, io.WriteCloser) (io.WriteCloser, error) {
				return nil, c.err
			})

			conn, err := RouteInFlight(ctx, router, astral.Launch(testQuery(t)))
			if !errors.Is(err, c.err) {
				t.Fatalf("want %v, got %v", c.err, err)
			}
			if conn != nil {
				t.Fatal("want no connection on error, got one")
			}
		})
	}
}

// Route is RouteInFlight plus a channel wrapper, for ops that speak objects
// rather than bytes.
func TestRoute_WrapsTheConnectionInAChannel(t *testing.T) {
	ctx := astral.NewContext(nil)

	router := routerFunc(func(*astral.Context, *astral.InFlightQuery, io.WriteCloser) (io.WriteCloser, error) {
		return nopWriteCloser{Writer: io.Discard}, nil
	})

	ch, err := Route(ctx, router, testQuery(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("want a channel, got nil")
	}
	ch.Close()
}

func TestRoute_PropagatesRouterErrors(t *testing.T) {
	ctx := astral.NewContext(nil)

	router := routerFunc(func(*astral.Context, *astral.InFlightQuery, io.WriteCloser) (io.WriteCloser, error) {
		return nil, astral.NewErrRouteNotFound()
	})

	ch, err := Route(ctx, router, testQuery(t))
	if !errors.Is(err, astral.NewErrRouteNotFound()) {
		t.Fatalf("want ErrRouteNotFound, got %v", err)
	}
	if ch != nil {
		t.Fatal("want no channel on error, got one")
	}
}

// Accept hands the responder an inbound Conn on a new goroutine, with the
// identities reversed relative to the caller's view.
func TestAccept_RunsTheHandlerWithAnInboundConn(t *testing.T) {
	q := testQuery(t)
	inFlight := astral.Launch(q)

	handled := make(chan astral.Conn, 1)

	writer, err := Accept(inFlight, nopWriteCloser{Writer: io.Discard}, func(conn astral.Conn) {
		handled <- conn
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writer == nil {
		t.Fatal("want a writer, got nil")
	}

	conn := <-handled

	if conn.(*Conn).Outbound() {
		t.Fatal("Outbound: want false for an accepted query")
	}
	if !conn.LocalIdentity().IsEqual(q.Target) {
		t.Fatal("LocalIdentity: want the target on the responder side")
	}
	if !conn.RemoteIdentity().IsEqual(q.Caller) {
		t.Fatal("RemoteIdentity: want the caller on the responder side")
	}
}

func TestReject_UsesTheDefaultCode(t *testing.T) {
	w, err := Reject()

	if w != nil {
		t.Fatal("want no writer, got one")
	}

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if rejected.Code != astral.DefaultRejectCode {
		t.Fatalf("Code: want %d, got %d", astral.DefaultRejectCode, rejected.Code)
	}
}

func TestRejectWithCode_CarriesTheCode(t *testing.T) {
	for _, code := range []uint8{1, 7, 255} {
		w, err := RejectWithCode(code)

		if w != nil {
			t.Fatal("want no writer, got one")
		}

		var rejected *astral.ErrRejected
		if !errors.As(err, &rejected) {
			t.Fatalf("want ErrRejected, got %v", err)
		}
		if rejected.Code != code {
			t.Fatalf("Code: want %d, got %d", code, rejected.Code)
		}
	}
}

// Code 0 is not a rejection — it is the "accepted" value on the wire, so
// rejecting with it is a programming error rather than a runtime one.
func TestRejectWithCode_PanicsOnZero(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("want a panic for code 0, got none")
		}
	}()

	RejectWithCode(0)
}

func TestRouteNotFound(t *testing.T) {
	w, err := RouteNotFound()

	if w != nil {
		t.Fatal("want no writer, got one")
	}
	if !errors.Is(err, astral.NewErrRouteNotFound()) {
		t.Fatalf("want ErrRouteNotFound, got %v", err)
	}
}
