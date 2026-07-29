package routing

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// NilRouter terminates a routing chain. The Soft flag picks which of the two
// terminal answers it gives, and the difference matters upstream: a
// PriorityRouter keeps walking past ErrRouteNotFound but stops on ErrRejected.

func TestNilRouter(t *testing.T) {
	cases := []struct {
		name    string
		soft    bool
		wantErr error
	}{
		{"hard rejects by default", false, &astral.ErrRejected{}},
		{"soft reports no route", true, &astral.ErrRouteNotFound{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := &NilRouter{Soft: c.soft}

			id := astral.GenerateIdentity()
			q := astral.Launch(query.New(id, id, "op", nil))

			w, err := router.RouteQuery(astral.NewContext(nil), q, nil)

			if w != nil {
				t.Fatal("want no writer, got one")
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("want %v, got %v", c.wantErr, err)
			}
		})
	}
}

// The hard rejection carries the default code, not an arbitrary one.
func TestNilRouter_HardRejectionUsesTheDefaultCode(t *testing.T) {
	id := astral.GenerateIdentity()
	q := astral.Launch(query.New(id, id, "op", nil))

	_, err := (&NilRouter{}).RouteQuery(astral.NewContext(nil), q, nil)

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if rejected.Code != astral.DefaultRejectCode {
		t.Fatalf("Code: want %d, got %d", astral.DefaultRejectCode, rejected.Code)
	}
}

var _ astral.Router = &NilRouter{}

var _ io.WriteCloser = (io.WriteCloser)(nil)
