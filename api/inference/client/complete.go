package inference

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/astral"
)

// ErrUnfinished reports an answer that ended before its last chunk: the
// channel closed, or eos came, while no chunk had named a FinishReason.
var ErrUnfinished = errors.New("inference: answer ended before its finish reason")

// Event is one item of an answer. Exactly one field is set.
type Event struct {
	Chunk *inference.Chunk
	Usage *inference.Usage
}

// Complete sends req to the target provider and yields the answer: chunks in
// Index order, then usage when the provider reports it. The sequence ends
// after eos, when the channel ends after the finishing chunk, or with one
// non-nil error:
//
//   - *inference.Error when the provider failed, before or after chunks;
//   - ErrUnfinished when the answer ended without a finishing chunk;
//   - a broken order: a chunk out of Index order or after the finishing
//     chunk, or usage before it or twice;
//   - the end of ctx;
//   - any other error of the query or the stream.
//
// The query channel closes when the sequence ends, when ctx ends, or when the
// caller stops ranging, and closing it cancels the request at the provider.
// Every range over the sequence sends the request again.
func (client *Client) Complete(ctx *astral.Context, req *inference.Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		ch, err := client.queryCh(ctx, inference.OpComplete, nil)
		if err != nil {
			yield(Event{}, err)
			return
		}
		defer ch.Close()

		// why: Receive does not watch ctx, so the end of ctx closes the
		// channel to end the read, which also tells the provider to stop.
		defer context.AfterFunc(ctx, func() { _ = ch.Close() })()

		if err := ch.Send(req); err != nil {
			yield(Event{}, ctxErr(ctx, err))
			return
		}

		var a answer
		for {
			obj, err := ch.Receive()
			if err != nil {
				if err = a.ended(err); err != nil {
					yield(Event{}, ctxErr(ctx, err))
				}
				return
			}
			ev, done, err := a.read(obj)
			switch {
			case err != nil:
				yield(Event{}, err)
				return
			case done:
				return
			case !yield(ev, nil):
				return
			}
		}
	}
}

// answer checks one answer against the stream rules as it arrives.
type answer struct {
	next     uint32
	finished bool
	usage    bool
}

// read turns one stream object into an event, and reports whether the answer
// is done.
func (a *answer) read(o astral.Object) (Event, bool, error) {
	switch obj := o.(type) {
	case *inference.Chunk:
		if err := a.chunk(obj); err != nil {
			return Event{}, false, err
		}
		return Event{Chunk: obj}, false, nil
	case *inference.Usage:
		if err := a.reported(); err != nil {
			return Event{}, false, err
		}
		return Event{Usage: obj}, false, nil
	case *astral.EOS:
		if !a.finished {
			return Event{}, false, ErrUnfinished
		}
		return Event{}, true, nil
	case error:
		return Event{}, false, obj
	}
	return Event{}, false, astral.NewErrUnexpectedObject(o)
}

func (a *answer) chunk(c *inference.Chunk) error {
	switch {
	case a.finished:
		return fmt.Errorf("inference: chunk %d after the finishing chunk", c.Index)
	case uint32(c.Index) != a.next:
		return fmt.Errorf("inference: chunk %d out of order, want %d", c.Index, a.next)
	}
	a.next++
	a.finished = c.FinishReason != ""
	return nil
}

// reported checks that usage follows the finishing chunk, once.
//
// why: a provider sends usage only for an answer that finished, and never
// after an inference.error, so usage that comes earlier belongs to an answer
// that broke off, and a caller would bill or show it as a whole one.
func (a *answer) reported() error {
	switch {
	case !a.finished:
		return errors.New("inference: usage before the finishing chunk")
	case a.usage:
		return errors.New("inference: usage reported twice")
	}
	a.usage = true
	return nil
}

// ended maps the read error that ended the channel. After the finishing
// chunk the answer is whole however the channel ends, EOF or a transport
// error: what came before tells (stream rules), and a client never waits for
// eos (op modes).
func (a *answer) ended(err error) error {
	switch {
	case a.finished:
		return nil
	case errors.Is(err, io.EOF):
		return ErrUnfinished
	}
	return err
}
