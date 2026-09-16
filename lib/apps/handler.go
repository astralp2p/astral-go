package apps

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	libapphost "github.com/astralp2p/astral-go/lib/apphost"
	"github.com/astralp2p/astral-go/lib/ipc"
)

// IntakeTimeout bounds the wait for a dialer's HandleQueryMsg. It covers the gap
// between connect and the first message only, never an established session, so a
// connection that opens and stays silent is dropped rather than held. A Handler
// takes its value when it is created.
var IntakeTimeout = 30 * time.Second

// Handler accepts inbound IPC queries from an apphost-registered endpoint.
// Close or context cancellation terminates all blocking calls.
type Handler struct {
	listener net.Listener
	ipcToken astral.Nonce
	doneCh   chan struct{}
	done     atomic.Bool
	queries  chan *PendingQuery
	// intakeTimeout is fixed at construction so intake never reads the package
	// default from its own goroutine.
	intakeTimeout time.Duration
	// acceptErr is written once by accept before it closes the Handler, so a
	// reader that has observed doneCh sees it.
	acceptErr atomic.Pointer[error]
}

// NewHandler creates an IPC listener with a random auth token.
// Registration with the node is the caller's responsibility.
func NewHandler() (*Handler, error) {
	return NewHandlerAt(libapphost.DefaultRouter().Protocol(), astral.NewNonce())
}

// NewHandlerAt creates an IPC listener on a system-assigned address for the given
// protocol and auth token.
func NewHandlerAt(protocol string, token astral.Nonce) (*Handler, error) {
	l, err := ipc.ListenAny(protocol)
	if err != nil {
		return nil, err
	}
	return newHandler(l, token), nil
}

// NewHandlerOn creates an IPC listener at the given "proto:addr" address and auth
// token. The node dials the address to deliver a query, so an address both sides
// name is what lets a node reach a handler across a filesystem or network boundary
// a system-assigned one does not cross.
func NewHandlerOn(ipcAddress string, token astral.Nonce) (*Handler, error) {
	l, err := ipc.Listen(ipcAddress)
	if err != nil {
		return nil, err
	}
	return newHandler(l, token), nil
}

func newHandler(l net.Listener, token astral.Nonce) *Handler {
	h := &Handler{
		listener:      l,
		doneCh:        make(chan struct{}),
		ipcToken:      token,
		queries:       make(chan *PendingQuery),
		intakeTimeout: IntakeTimeout,
	}

	go h.accept()

	return h
}

// ReadQuery waits for and returns the next pending query
func (h *Handler) ReadQuery() (*PendingQuery, error) {
	select {
	case pending := <-h.queries:
		return pending, nil

	case <-h.doneCh:
		if err := h.acceptErr.Load(); err != nil {
			return nil, *err
		}
		return nil, net.ErrClosed
	}
}

// accept takes connections off the listener and screens each one in its own
// goroutine. Screening blocks on the dialer, so doing it here would let one
// silent dialer stall every other inbound query.
func (h *Handler) accept() {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			h.acceptErr.Store(&err)
			h.Close()
			return
		}

		go h.intake(conn)
	}
}

// intake reads the dialer's HandleQueryMsg, authenticates it, and hands the query
// to ReadQuery. A connection that fails any step is answered and closed here.
func (h *Handler) intake(conn net.Conn) {
	ch := channel.New(conn)

	// bound the wait for the first message; an unsent one must not hold the conn
	_ = conn.SetReadDeadline(time.Now().Add(h.intakeTimeout))

	obj, err := ch.Receive()
	if err != nil {
		ch.Close()
		return
	}

	// check message type - must be HandleQueryMsg
	queryMsg, ok := obj.(*apphost.HandleQueryMsg)
	if !ok {
		ch.Send(&apphost.ErrorMsg{Code: apphost.ErrCodeProtocolError})
		ch.Close()
		return
	}

	// check auth ipcToken
	if queryMsg.IPCToken != h.ipcToken {
		ch.Send(&apphost.ErrorMsg{Code: apphost.ErrCodeDenied})
		ch.Close()
		return
	}

	// the session that follows is not bounded by the intake deadline
	_ = conn.SetReadDeadline(time.Time{})

	pending := &PendingQuery{
		conn: conn,
		query: &astral.Query{
			Nonce:       queryMsg.ID,
			Caller:      queryMsg.Caller,
			Target:      queryMsg.Target,
			QueryString: astral.String32(queryMsg.Query),
		},
	}

	select {
	case h.queries <- pending:
	case <-h.doneCh:
		conn.Close()
	}
}

type HandleFunc func(ctx *astral.Context, query *PendingQuery) error

// Serve calls the given HandleFunc for every query received
func (h *Handler) Serve(ctx *astral.Context, handle HandleFunc) error {
	stop := h.closeOnDone(ctx)
	defer stop()

	for {
		pending, err := h.ReadQuery()
		if err != nil {
			if ctx.Err() != nil && isClosedListenerErr(err) {
				return ctx.Err()
			}
			return err
		}

		err = handle(ctx, pending)
		if err != nil {
			return err
		}
	}
}

// Route routes every query to given astral.Router
func (h *Handler) Route(ctx *astral.Context, router astral.Router) error {
	var errRejected *astral.ErrRejected

	stop := h.closeOnDone(ctx)
	defer stop()

	for {
		// get the next pending query
		pending, err := h.ReadQuery()
		if err != nil {
			if ctx.Err() != nil && isClosedListenerErr(err) {
				return ctx.Err()
			}
			return err
		}

		// lock the writer so that we can send the query response before the target starts sending data
		lockedWriter := newLockableWriteCloser(pending.conn)
		lockedWriter.Lock()

		// route the query to the target
		w, err := router.RouteQuery(ctx, astral.Launch(pending.query), lockedWriter)
		switch {
		case err == nil:
			// accepted - send an Ack and release the writer to the query target
			conn := pending.Accept()
			lockedWriter.Unlock()

			// forward the traffic
			go func() {
				io.Copy(w, conn)
				w.Close()
			}()
		case errors.As(err, &errRejected):
			// rejected - forward the rejection code and release the writer
			pending.RejectWithCode(int(errRejected.Code))
			lockedWriter.Unlock()

		default:
			pending.Skip()
			lockedWriter.Unlock()
		}
	}
}

// Token returns the token expected by the Handler
func (h *Handler) Token() astral.Nonce {
	return h.ipcToken
}

func (h *Handler) Close() error {
	if h.done.CompareAndSwap(false, true) {
		close(h.doneCh)
	}
	return h.listener.Close()
}

func (h *Handler) closeOnDone(ctx *astral.Context) func() {
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			_ = h.Close()
		case <-done:
		}
	}()

	return func() { close(done) }
}

// todo: weird error checking - look for better solution
func isClosedListenerErr(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "connection closed")
}

func (h *Handler) String() string {
	return h.Endpoint()
}

func (h *Handler) Endpoint() string {
	a := h.listener.Addr()
	return a.Network() + ":" + a.String()
}

// Done returns a channel that will be closed when the Handler is closed
func (h *Handler) Done() <-chan struct{} {
	return h.doneCh
}
