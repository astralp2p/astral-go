package log

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	alog "github.com/astralp2p/astral-go/astral/log"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

// fakeRouter records the query it routes and answers with pre-encoded objects.
type fakeRouter struct {
	queryString string
	answer      *bytes.Buffer
}

func (r *fakeRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	r.queryString = q.QueryString.String()
	return query.NewConn(nil, nil, nopWriteCloser{io.Discard}, r.answer, true), nil
}

func (r *fakeRouter) GuestID() *astral.Identity { return nil }
func (r *fakeRouter) HostID() *astral.Identity  { return nil }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func answeringClient(t *testing.T, objects ...astral.Object) (*Client, *fakeRouter) {
	t.Helper()

	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	router := &fakeRouter{answer: &buf}
	return New(nil, astrald.New(router)), router
}

// The literal is spelled out rather than taken from MethodListen, because a
// test that reads the same constant as the code cannot catch the constant
// being wrong.
func TestListen_NamesTheOp(t *testing.T) {
	c, router := answeringClient(t)

	l, err := c.Listen(astral.NewContext(nil))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer l.Close()

	if path, _ := query.Parse(router.queryString); path != "log.listen" {
		t.Fatalf("path: want log.listen, got %q", path)
	}
}

// An entry carrying an object type without a blueprint arrives as an
// UnparsedObject, and the entries after it still decode.
func TestListen_UnknownObjectTypeKeepsTheStream(t *testing.T) {
	first := alog.NewEntry(nil, 1, astral.NewString32("first"))
	unknown := alog.NewEntry(nil, 1, astral.NewUnparsedObject("test.no_blueprint", []byte{1, 2, 3}))
	last := alog.NewEntry(nil, 2, astral.NewString32("last"))

	c, _ := answeringClient(t, first, unknown, last)
	l, err := c.Listen(astral.NewContext(nil))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer l.Close()

	if e := nextEntry(t, l); e.Level != 1 || len(e.Objects) != 1 {
		t.Fatalf("first: got %+v", e)
	}

	obj, err := l.Next()
	if err != nil {
		t.Fatalf("unknown: %v", err)
	}
	u, ok := obj.(*astral.UnparsedObject)
	if !ok || u.ObjectType() != "astrald.log.entry" {
		t.Fatalf("unknown: want an unparsed astrald.log.entry, got %T %v", obj, obj)
	}

	if e := nextEntry(t, l); e.Level != 2 {
		t.Fatalf("last: got %+v", e)
	}

	if _, err := l.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("end: want io.EOF, got %v", err)
	}
}

func TestListen_ErrorObjectIsReturnedAsError(t *testing.T) {
	c, _ := answeringClient(t, astral.NewError("denied"))
	l, err := c.Listen(astral.NewContext(nil))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer l.Close()

	if _, err := l.Next(); err == nil || err.Error() != "denied" {
		t.Fatalf("want error denied, got %v", err)
	}
}

func nextEntry(t *testing.T, l *Listener) *alog.Entry {
	t.Helper()

	obj, err := l.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	e, ok := obj.(*alog.Entry)
	if !ok {
		t.Fatalf("Next: want *log.Entry, got %T", obj)
	}
	return e
}
