package apphost

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

// hostsRouter records the query it routes and answers with pre-encoded objects.
type hostsRouter struct {
	queryString string
	answer      io.Reader
}

func (r *hostsRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	r.queryString = q.QueryString.String()
	return query.NewConn(nil, nil, nopWriteCloser{io.Discard}, r.answer, true), nil
}

func (r *hostsRouter) GuestID() *astral.Identity { return nil }
func (r *hostsRouter) HostID() *astral.Identity  { return nil }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func hostsClient(t *testing.T, objects ...astral.Object) (*Client, *hostsRouter) {
	t.Helper()

	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	router := &hostsRouter{answer: &buf}
	return New(nil, astrald.New(router)), router
}

func TestHosts_StreamsIdentitiesUntilEOS(t *testing.T) {
	a, b := astral.GenerateIdentity(), astral.GenerateIdentity()
	client, router := hostsClient(t, a, b, &astral.EOS{})

	out, errPtr := client.Hosts(astral.NewContext(nil), "chat")

	var got []*astral.Identity
	for id := range out {
		got = append(got, id)
	}

	if *errPtr != nil {
		t.Fatalf("Hosts: %v", *errPtr)
	}

	path, params := query.Parse(router.queryString)
	if path != apphost.MethodHosts {
		t.Fatalf("path: want %v, got %q", apphost.MethodHosts, path)
	}
	if len(params) != 1 || params["app"] != "chat" {
		t.Fatalf("params: want app=chat, got %v", params)
	}

	if len(got) != 2 || !got[0].IsEqual(a) || !got[1].IsEqual(b) {
		t.Fatalf("hosts: want [%v %v], got %v", a, b, got)
	}
}

func TestHosts_EOSAloneIsNoHosts(t *testing.T) {
	client, _ := hostsClient(t, &astral.EOS{})

	out, errPtr := client.Hosts(astral.NewContext(nil), "nobody")

	for id := range out {
		t.Fatalf("want no hosts, got %v", id)
	}
	if *errPtr != nil {
		t.Fatalf("Hosts: %v", *errPtr)
	}
}

func TestHosts_ErrorMessageIsReturned(t *testing.T) {
	client, _ := hostsClient(t, astral.Err(errors.New("db down")))

	out, errPtr := client.Hosts(astral.NewContext(nil), "chat")

	for id := range out {
		t.Fatalf("want no hosts, got %v", id)
	}
	if *errPtr == nil || (*errPtr).Error() != "db down" {
		t.Fatalf("want error %q, got %v", "db down", *errPtr)
	}
}
