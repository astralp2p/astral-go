package indexing

import (
	"bytes"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/indexing"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
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

// The literals are spelled out rather than taken from MethodEnableRepo, because
// a test that reads the same constant as the code cannot catch the constant
// being wrong. astrald's opEnableRepoArgs declares Repo and Disable.
func TestEnableRepo_NamesTheOpAndArgumentsTheNodeReads(t *testing.T) {
	c, router := answeringClient(t, &astral.Ack{})

	if err := c.EnableRepo(astral.NewContext(nil), "local"); err != nil {
		t.Fatalf("EnableRepo: %v", err)
	}

	path, params := query.Parse(router.queryString)
	if path != "indexing.enable_repo" {
		t.Fatalf("path: want indexing.enable_repo, got %q", path)
	}
	if params["repo"] != "local" {
		t.Fatalf("repo: want local, got %q in %v", params["repo"], params)
	}
	if params["disable"] != "false" {
		t.Fatalf("disable: want false, got %q in %v", params["disable"], params)
	}
}

func TestDisableRepo_SetsDisable(t *testing.T) {
	c, router := answeringClient(t, &astral.Ack{})

	if err := c.DisableRepo(astral.NewContext(nil), "local"); err != nil {
		t.Fatalf("DisableRepo: %v", err)
	}

	path, params := query.Parse(router.queryString)
	if path != "indexing.enable_repo" {
		t.Fatalf("path: want indexing.enable_repo, got %q", path)
	}
	if params["repo"] != "local" || params["disable"] != "true" {
		t.Fatalf("want repo=local disable=true, got %v", params)
	}
}

// An unknown repository arrives as an error object, not as an ack.
func TestEnableRepo_NodeErrorIsReturned(t *testing.T) {
	c, _ := answeringClient(t, astral.Err(indexing.ErrRepositoryNotFound))

	err := c.EnableRepo(astral.NewContext(nil), "missing")
	if err == nil {
		t.Fatal("expected the node's error to surface")
	}
	if err.Error() != indexing.ErrRepositoryNotFound.Error() {
		t.Fatalf("want %q, got %q", indexing.ErrRepositoryNotFound, err)
	}
}
