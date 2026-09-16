package apps

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/ipc"
)

// newTestHandler binds a handler on a private unix socket and returns it with its
// address, so a test can dial the handler the way a node does.
func newTestHandler(t *testing.T) (*Handler, string, astral.Nonce) {
	t.Helper()

	address := "unix:" + filepath.Join(t.TempDir(), "handler.sock")
	token := astral.NewNonce()

	h, err := NewHandlerOn(address, token)
	if err != nil {
		t.Fatalf("new handler on %v: %v", address, err)
	}
	t.Cleanup(func() { h.Close() })

	return h, address, token
}

// dialQuery delivers one HandleQueryMsg, as the node does when routing an inbound
// query to a registered handler.
func dialQuery(t *testing.T, address string, token astral.Nonce, query string) {
	t.Helper()

	conn, err := ipc.Dial(address)
	if err != nil {
		t.Fatalf("dial %v: %v", address, err)
	}
	t.Cleanup(func() { conn.Close() })

	err = channel.New(conn).Send(&apphost.HandleQueryMsg{
		IPCToken: token,
		ID:       astral.NewNonce(),
		Caller:   astral.Anyone,
		Target:   astral.Anyone,
		Query:    astral.String16(query),
	})
	if err != nil {
		t.Fatalf("send handle_query_msg: %v", err)
	}
}

// readQuery returns what ReadQuery produced, or fails the test if it produced
// nothing before the deadline.
func readQuery(t *testing.T, h *Handler, within time.Duration) *PendingQuery {
	t.Helper()

	type result struct {
		pending *PendingQuery
		err     error
	}
	done := make(chan result, 1)

	go func() {
		pending, err := h.ReadQuery()
		done <- result{pending, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("read query: %v", r.err)
		}
		t.Cleanup(func() { r.pending.Close() })
		return r.pending

	case <-time.After(within):
		t.Fatal("ReadQuery returned nothing before the deadline")
		return nil
	}
}

// TestReadQuery_survivesASilentDialer is the defect this file exists for: screening
// a dialer used to happen on the accept loop, so one connection that opened and
// sent nothing stalled every query behind it for as long as it stayed open.
func TestReadQuery_survivesASilentDialer(t *testing.T) {
	h, address, token := newTestHandler(t)

	// connect and send nothing, holding the connection open for the whole test
	silent, err := ipc.Dial(address)
	if err != nil {
		t.Fatalf("dial %v: %v", address, err)
	}
	defer silent.Close()

	dialQuery(t, address, token, "behind-the-silent-one")

	if got := readQuery(t, h, 5*time.Second).Query(); got != "behind-the-silent-one" {
		t.Fatalf("query: got %v, want behind-the-silent-one", got)
	}
}

// TestReadQuery_survivesAnUnauthenticatedDialer: the token is checked after the
// blocking read, so a dialer needs no token to reach the screening step. It must
// not reach the queries behind it either.
func TestReadQuery_survivesAnUnauthenticatedDialer(t *testing.T) {
	h, address, token := newTestHandler(t)

	dialQuery(t, address, astral.NewNonce(), "rejected")
	dialQuery(t, address, token, "accepted")

	if got := readQuery(t, h, 5*time.Second).Query(); got != "accepted" {
		t.Fatalf("query: got %v, want accepted", got)
	}
}

// TestIntake_dropsASilentDialer: the intake deadline covers the gap before the
// first message, so a silent connection is released instead of held forever.
func TestIntake_dropsASilentDialer(t *testing.T) {
	previous := IntakeTimeout
	IntakeTimeout = 100 * time.Millisecond
	t.Cleanup(func() { IntakeTimeout = previous })

	_, address, _ := newTestHandler(t)

	conn, err := ipc.Dial(address)
	if err != nil {
		t.Fatalf("dial %v: %v", address, err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// the handler closing its side ends this read; without the deadline it blocks
	var buf [1]byte
	if _, err = conn.Read(buf[:]); err == nil {
		t.Fatal("want the handler to have closed the silent connection")
	}
}

// TestReadQuery_reportsTheListenerClosing keeps Serve and Route able to tell a
// shutdown from a failure: both test the error with isClosedListenerErr.
func TestReadQuery_reportsTheListenerClosing(t *testing.T) {
	h, _, _ := newTestHandler(t)

	h.Close()

	_, err := h.ReadQuery()
	if err == nil {
		t.Fatal("want an error once the handler is closed")
	}
	if !isClosedListenerErr(err) {
		t.Fatalf("error %v is not recognised as the listener closing", err)
	}
}
