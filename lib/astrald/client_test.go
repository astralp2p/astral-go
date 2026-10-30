package astrald

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
	libapphost "github.com/astralp2p/astral-go/lib/apphost"
)

// hungHost greets every session and then never answers: the token, a route
// request, anything. It returns the endpoint to dial.
func hungHost(t *testing.T) string {
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
			go func() {
				if channel.New(c).Send(&apphost.HostInfoMsg{Identity: astral.GenerateIdentity()}) == nil {
					_, _ = io.Copy(io.Discard, c)
				}
			}()
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

type openGate struct{}

func (openGate) Ready() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// A query learns its caller before it is sent. A host that holds the token
// unanswered must not hold the query past its ctx, through lib/apphost's
// Router and through each wrapper of it.
func TestQueryEndsWithCtxWhileTheHostHoldsTheToken(t *testing.T) {
	endpoint := hungHost(t)
	routers := map[string]func() Router{
		"apphost": func() Router { return libapphost.NewRouter(endpoint, "token") },
		"retry":   func() Router { return NewNoRetryRouter(libapphost.NewRouter(endpoint, "token")) },
		"monitor": func() Router { return NewRouterMonitor(libapphost.NewRouter(endpoint, "token")) },
		"gate":    func() Router { return NewGateRouter(libapphost.NewRouter(endpoint, "token"), openGate{}) },
	}
	for name, router := range routers {
		t.Run(name, func(t *testing.T) {
			client := New(router())
			ctx, cancel := astral.NewContext(nil).WithTimeout(100 * time.Millisecond)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				conn, err := client.Query(ctx, "test.op", nil)
				if conn != nil {
					conn.Close()
				}
				done <- err
			}()

			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("err = %v; want context.DeadlineExceeded", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Query still waiting on the host 5 s after ctx ended")
			}
		})
	}
}
