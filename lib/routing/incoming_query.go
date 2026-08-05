package routing

import (
	"errors"
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// IncomingQuery is a server-side view of an in-flight query that must be
// resolved exactly once via Accept* or Reject* before the 5-second deadline
// expires or the context is cancelled.
type IncomingQuery struct {
	*astral.Query
	origin       string
	remoteWriter io.WriteCloser
	response     chan queryResponse
	resolved     atomic.Bool

	// mu guards accepted, which holds the read half AcceptRaw kept for the op. It is
	// retained only so a panicking op's abandoned connection can be torn down, and is
	// nil for a query that was never accepted. AcceptRaw normally runs on the same
	// goroutine as the teardown, but an op may accept from one it spawned, so the two
	// are synchronised rather than assumed ordered.
	mu       sync.Mutex
	accepted *io.PipeReader
}

var typeOfIncomingQuery = reflect.TypeOf((*IncomingQuery)(nil)).Elem()

func NewIncomingQuery(query *astral.Query, remoteWriter io.WriteCloser, origin string) *IncomingQuery {
	return &IncomingQuery{
		remoteWriter: remoteWriter,
		Query:        query,
		origin:       origin,
		response:     make(chan queryResponse, 1),
	}
}

// AcceptRaw accepts the query and returns a raw binary connection
func (query *IncomingQuery) AcceptRaw() (conn io.ReadWriteCloser) {
	if !query.resolved.CompareAndSwap(false, true) {
		return ErrorConn{Err: errors.New("query already resolved")}
	}

	localReader, localWriter := io.Pipe()

	query.mu.Lock()
	query.accepted = localReader
	query.mu.Unlock()

	conn = NewConn(query.Query.Caller, query.Query.Target, query.remoteWriter, localReader, false)

	query.response <- queryResponse{localWriter, nil}

	return conn
}

// abandon tears down a connection whose op accepted it and then failed to service it.
//
// why: AcceptRaw gives the caller the write half of a pipe and keeps the read half for
// the op. A pipe write parks until a reader takes the bytes, so once the op's goroutine
// is gone the caller's first write never returns — not a timeout, an indefinite block.
// Closing the read half with the cause turns that block into an error naming it, and
// closing remoteWriter ends the response direction so the caller sees a finished stream
// rather than silence.
//
// Noop unless the query was accepted: a rejected query has no connection to tear down,
// so calling this after RejectWithCode is safe and does nothing.
func (query *IncomingQuery) abandon(cause error) {
	query.mu.Lock()
	r := query.accepted
	query.accepted = nil
	query.mu.Unlock()

	if r == nil {
		return
	}

	r.CloseWithError(cause)

	if query.remoteWriter != nil {
		query.remoteWriter.Close()
	}
}

// Accept accepts the query and returns the established channel
func (query *IncomingQuery) Accept(cfg ...channel.ConfigFunc) *channel.Channel {
	return channel.New(query.AcceptRaw(), cfg...)
}

// Reject rejects the query with the default reject code
func (query *IncomingQuery) Reject() (err error) {
	if !query.resolved.CompareAndSwap(false, true) {
		return errors.New("query already resolved")
	}

	err = &astral.ErrRejected{Code: astral.DefaultRejectCode}
	query.response <- queryResponse{nil, err}

	return nil
}

// RejectWithCode rejects the query with the provided reject code
func (query *IncomingQuery) RejectWithCode(code uint8) (err error) {
	if !query.resolved.CompareAndSwap(false, true) {
		return errors.New("query already resolved")
	}

	if code == 0 {
		code = 1
	}

	err = astral.NewErrRejected(code)
	query.response <- queryResponse{nil, err}

	return nil
}

// Caller returns the caller identity
func (query *IncomingQuery) Caller() *astral.Identity {
	return query.Query.Caller
}

// Target returns the target identity
func (query *IncomingQuery) Target() *astral.Identity {
	return query.Query.Target
}

// QueryString returns the full query string
func (query *IncomingQuery) QueryString() string { return query.Query.QueryString }

// Nonce returns the query nonce
func (query *IncomingQuery) Nonce() astral.Nonce {
	return query.Query.Nonce
}

// Origin returns the query origin (empty for local, "network" for network queries)
func (query *IncomingQuery) Origin() string {
	return query.origin
}

// await waits for the query to resolve and returns the result. Can only be called once.
func (query *IncomingQuery) await(ctx *astral.Context) (io.WriteCloser, error) {
	select {
	case r := <-query.response:
		return r.WriteCloser, r.Error

	case <-ctx.Done():
		query.Reject()
		return nil, astral.NewErrRejected(astral.CodeCanceled)

	case <-time.After(5 * time.Second):
		query.Reject()
		return nil, astral.NewErrRejected(0)
	}
}

type queryResponse struct {
	io.WriteCloser
	Error error
}
