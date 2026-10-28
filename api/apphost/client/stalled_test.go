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
	libapphost "github.com/astralp2p/astral-go/lib/apphost"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// A stalled node accepts a query and never answers it, as a node does while a
// delegated register policy waits on a person, or one that hangs past the
// accept. It reads all the caller sends.
//
// stalledRouters reaches one over each transport a caller meets: in-process
// routing, and a socket conn of the type lib/apphost.Router returns. A
// channel closed under a read fails differently on each: in-process it reads
// as EOF, on a socket as a closed connection.
func stalledRouters(t *testing.T, methods ...string) map[string]astrald.Router {
	t.Helper()
	return map[string]astrald.Router{
		"in-process": stalledOps(t, methods),
		"socket":     stalledSocket(t),
	}
}

func stalledOps(t *testing.T, methods []string) astrald.Router {
	t.Helper()
	router := routing.NewOpRouter()
	for _, method := range methods {
		op, err := routing.NewOp(func(_ *astral.Context, q *routing.IncomingQuery) error {
			conn := q.AcceptRaw()
			_, _ = io.Copy(io.Discard, conn)
			return conn.Close()
		})
		if err != nil {
			t.Fatalf("NewOp %s: %v", method, err)
		}
		if err := router.AddOp(method, op); err != nil {
			t.Fatalf("AddOp %s: %v", method, err)
		}
	}
	return &opsRouter{inner: router}
}

// opsRouter routes queries in-process to the ops of inner.
type opsRouter struct{ inner astral.Router }

func (r *opsRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (*opsRouter) GuestID() *astral.Identity { return nil }
func (*opsRouter) HostID() *astral.Identity  { return nil }

// socketRouter connects every query to a listener that reads and never writes.
type socketRouter struct{ addr string }

func stalledSocket(t *testing.T) astrald.Router {
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
			go func() { _, _ = io.Copy(io.Discard, c) }()
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
	return &socketRouter{addr: ln.Addr().String()}
}

func (r *socketRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	c, err := net.Dial("tcp", r.addr)
	if err != nil {
		return nil, err
	}
	return libapphost.NewConn(c, q.Query, true), nil
}

func (*socketRouter) GuestID() *astral.Identity { return nil }
func (*socketRouter) HostID() *astral.Identity  { return nil }

// endsWithCtx runs call with a ctx that times out while the node is silent,
// and fails unless call returns that ctx's error soon after.
func endsWithCtx(t *testing.T, call func(*astral.Context) error) {
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
	case <-time.After(5 * time.Second):
		t.Fatal("still waiting on the node 5 s after ctx ended")
	}
}

// The node accepts apphost.register before its policy decides, so only ctx
// ends a wait for a policy that never answers.
func TestRegisterWith_EndsWithCtx(t *testing.T) {
	for name, router := range stalledRouters(t, apphost.MethodRegister) {
		t.Run(name, func(t *testing.T) {
			client := New(nil, astrald.New(router))
			endsWithCtx(t, func(ctx *astral.Context) error {
				_, err := client.RegisterWith(ctx, RegisterOptions{Name: "app", GrantPermits: []string{"mod.user.see_swarm_action"}})
				return err
			})
		})
	}
}

// A caller tells the end of its own ctx from a node that answered wrong.
func TestWhoami_EndsWithCtx(t *testing.T) {
	for name, router := range stalledRouters(t, apphost.MethodWhoami) {
		t.Run(name, func(t *testing.T) {
			client := New(nil, astrald.New(router))
			endsWithCtx(t, func(ctx *astral.Context) error {
				_, err := client.Whoami(ctx)
				return err
			})
		})
	}
}

func TestGrant_EndsWithCtx(t *testing.T) {
	for name, router := range stalledRouters(t, apphost.MethodGrant) {
		t.Run(name, func(t *testing.T) {
			client := New(nil, astrald.New(router))
			endsWithCtx(t, func(ctx *astral.Context) error {
				return client.Grant(ctx, astral.GenerateIdentity(), "mod.user.see_swarm_action", GrantOptions{})
			})
		})
	}
}
