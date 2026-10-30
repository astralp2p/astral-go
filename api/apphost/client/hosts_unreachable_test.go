package apphost

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	libapphost "github.com/astralp2p/astral-go/lib/apphost"
	"github.com/astralp2p/astral-go/lib/astrald"
)

// unreachableRouter fails every query before it reaches a node.
type unreachableRouter struct{ err error }

func (r unreachableRouter) RouteQuery(*astral.Context, *astral.InFlightQuery) (astral.Conn, error) {
	return nil, r.err
}

func (unreachableRouter) GuestID() *astral.Identity { return nil }
func (unreachableRouter) HostID() *astral.Identity  { return nil }

// rangeHosts ranges over out as a caller does, and fails unless the range
// ends.
func rangeHosts(t *testing.T, out <-chan *astral.Identity) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for id := range out {
			t.Errorf("want no hosts, got %v", id)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the range still waits 5 s after the query failed")
	}
}

// A caller ranges over the hosts and then reads the error, so a query that
// fails must still end the range.
func TestHosts_FailedQueryClosesTheChannel(t *testing.T) {
	want := errors.New("node unavailable")
	client := New(nil, astrald.New(unreachableRouter{want}))

	out, errPtr := client.Hosts(astral.NewContext(nil), "chat")
	rangeHosts(t, out)

	if !errors.Is(*errPtr, want) {
		t.Fatalf("want %v, got %v", want, *errPtr)
	}
}

// An endpoint that accepts and never speaks fails the query when ctx ends,
// and the range ends with it.
func TestHosts_SilentNodeEndsWithCtx(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	conns := make(chan net.Conn, 8)
	t.Cleanup(func() {
		_ = ln.Close()
		for len(conns) > 0 {
			_ = (<-conns).Close()
		}
	})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			conns <- c
			go func() { _, _ = io.Copy(io.Discard, c) }()
		}
	}()
	client := New(nil, astrald.New(libapphost.NewRouter("tcp:"+ln.Addr().String(), "token")))
	ctx, cancel := astral.NewContext(nil).WithTimeout(100 * time.Millisecond)
	defer cancel()

	out, errPtr := client.Hosts(ctx, "chat")
	rangeHosts(t, out)

	if !errors.Is(*errPtr, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", *errPtr)
	}
}
