package apphost

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

// recordingRouter records the query it routes and what the client sends, and
// answers with pre-encoded objects.
type recordingRouter struct {
	queryString string
	sent        bytes.Buffer
	answer      io.Reader
}

func (r *recordingRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	r.queryString = q.QueryString.String()
	return query.NewConn(nil, nil, nopWriteCloser{&r.sent}, r.answer, true), nil
}

func (r *recordingRouter) GuestID() *astral.Identity { return nil }
func (r *recordingRouter) HostID() *astral.Identity  { return nil }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// recordingClient returns a client whose node answers with objects.
func recordingClient(t *testing.T, objects ...astral.Object) (*Client, *recordingRouter) {
	t.Helper()

	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	router := &recordingRouter{answer: &buf}
	return New(nil, astrald.New(router)), router
}

// decodeAll decodes the objects in b, as the node reads what the client sent.
func decodeAll(t *testing.T, b []byte) []astral.Object {
	t.Helper()

	r := channel.NewReceiver(bytes.NewReader(b))
	var list []astral.Object
	for {
		o, err := r.Receive()
		if errors.Is(err, io.EOF) {
			return list
		}
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		list = append(list, o)
	}
}
