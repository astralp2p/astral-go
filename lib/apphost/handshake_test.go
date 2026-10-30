package apphost

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// fakeHost listens on a loopback port and serves every connection with serve.
// It returns the endpoint to dial.
func fakeHost(t *testing.T, serve func(net.Conn)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	var mu sync.Mutex
	var conns []net.Conn
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
			go serve(c)
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})
	return "tcp:" + ln.Addr().String()
}

// silent accepts and never speaks: a port that is not an apphost endpoint.
func silent(c net.Conn) { _, _ = io.Copy(io.Discard, c) }

// greeting greets as host and then never answers: a host that hangs after
// the greeting.
func greeting(host *astral.Identity) func(net.Conn) {
	return func(c net.Conn) {
		if err := channel.New(c).Send(&apphost.HostInfoMsg{Identity: host, Alias: "node"}); err != nil {
			return
		}
		silent(c)
	}
}

// answering greets as host and answers every token with guest.
func answering(host, guest *astral.Identity) func(net.Conn) {
	return func(c net.Conn) {
		ch := channel.New(c)
		if err := ch.Send(&apphost.HostInfoMsg{Identity: host, Alias: "node"}); err != nil {
			return
		}
		for {
			msg, err := ch.Receive()
			if err != nil {
				return
			}
			if _, ok := msg.(*apphost.AuthTokenMsg); ok {
				_ = ch.Send(&apphost.AuthSuccessMsg{GuestID: guest})
			}
		}
	}
}

// endsWithCtx runs call with a ctx that times out while the host is silent,
// and fails unless call returns that ctx's error soon after.
func endsWithCtx(t *testing.T, call func(*astral.Context) error) error {
	t.Helper()
	ctx, cancel := astral.NewContext(nil).WithTimeout(100 * time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- call(ctx) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v; want context.DeadlineExceeded", err)
		}
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("still waiting on the host 5 s after ctx ended")
	}
	return nil
}

func TestConnectEndsWithCtxWhenTheHostNeverGreets(t *testing.T) {
	endpoint := fakeHost(t, silent)
	endsWithCtx(t, func(ctx *astral.Context) error {
		_, err := Connect(ctx, endpoint)
		return err
	})
}

// A session that opened is the caller's: the end of the ctx it opened with
// does not close it.
func TestConnectLeavesTheSessionToTheCaller(t *testing.T) {
	host, guest := astral.GenerateIdentity(), astral.GenerateIdentity()
	endpoint := fakeHost(t, answering(host, guest))
	ctx, cancel := astral.NewContext(nil).WithCancel()

	s, err := Connect(ctx, endpoint)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer s.Close()
	cancel()

	if !s.HostID().IsEqual(host) {
		t.Fatalf("host %v; want %v", s.HostID(), host)
	}
	if err := s.AuthToken("token"); err != nil || !s.GuestID().IsEqual(guest) {
		t.Fatalf("AuthToken after ctx ended: %v, guest %v", err, s.GuestID())
	}
}

// The router's handshake, auth included, ends with the ctx it runs in, and
// says the node was unavailable.
func TestRouterHandshakeEndsWithCtx(t *testing.T) {
	calls := map[string]func(*Router, *astral.Context) error{
		"GuestIDContext": func(r *Router, ctx *astral.Context) error { _, err := r.GuestIDContext(ctx); return err },
		"HostIDContext":  func(r *Router, ctx *astral.Context) error { _, err := r.HostIDContext(ctx); return err },
		"RouteQuery": func(r *Router, ctx *astral.Context) error {
			_, err := r.RouteQuery(ctx, astral.Launch(query.New(nil, nil, "test.op", nil)))
			return err
		},
	}
	for host, serve := range map[string]func(net.Conn){
		"no greeting":     silent,
		"no token answer": greeting(astral.GenerateIdentity()),
	} {
		endpoint := fakeHost(t, serve)
		for name, call := range calls {
			t.Run(host+"/"+name, func(t *testing.T) {
				// why a router each: a greeting caches the host's identity
				router := NewRouter(endpoint, "token")
				err := endsWithCtx(t, func(ctx *astral.Context) error { return call(router, ctx) })
				if !errors.Is(err, ErrNodeUnavailable) {
					t.Fatalf("err = %v; want ErrNodeUnavailable", err)
				}
			})
		}
	}
}

func TestRouterIdentitiesWithinCtx(t *testing.T) {
	host, guest := astral.GenerateIdentity(), astral.GenerateIdentity()
	router := NewRouter(fakeHost(t, answering(host, guest)), "token")
	ctx, cancel := astral.NewContext(nil).WithTimeout(5 * time.Second)
	defer cancel()

	if id, err := router.GuestIDContext(ctx); err != nil || !id.IsEqual(guest) {
		t.Fatalf("GuestIDContext = %v, %v; want %v", id, err, guest)
	}
	if id, err := router.HostIDContext(ctx); err != nil || !id.IsEqual(host) {
		t.Fatalf("HostIDContext = %v, %v; want %v", id, err, host)
	}
}

// A router without a token has no guest identity to learn, so it does not
// dial for one.
func TestRouterWithoutTokenHasNoGuest(t *testing.T) {
	router := NewRouter("tcp:127.0.0.1:1", "")
	if id, err := router.GuestIDContext(astral.NewContext(nil)); id != nil || err != nil {
		t.Fatalf("GuestIDContext = %v, %v; want nil, nil", id, err)
	}
}
