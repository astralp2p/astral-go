package routing

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// UntilDisconnect returns a context that ends when the requester's side of
// conn ends, and a stop func the op defers. It suits an op whose requester
// sends nothing more once the watch starts, so a read returns only when the
// requester closes its side.
//
// why: an op runs on a context detached from the requester's, so nothing else
// tells a waiting decision that nobody waits for its answer.
//
// why the reader beneath Conn: Conn.Read closes the reply direction on any
// read error, so a watch through it would cut the answer off from a requester
// that only closed its write side.
//
// note: a requester that closes its write side before the answer arrives
// counts as having left. stop closes the read side, which ends the watch when
// the op returns before the requester closes.
func UntilDisconnect(ctx *astral.Context, conn io.ReadWriteCloser) (*astral.Context, func()) {
	ctx, cancel := ctx.WithCancel()

	var r io.Reader = conn
	var readSide io.Closer
	if c, ok := conn.(*Conn); ok {
		r = c.Reader
		readSide, _ = c.Reader.(io.Closer)
	}

	go func() {
		defer cancel()
		_, _ = io.Copy(io.Discard, r)
	}()

	return ctx, func() {
		cancel()
		if readSide != nil {
			_ = readSide.Close()
		}
	}
}
