package channel

import (
	"github.com/astralp2p/astral-go/astral"
)

// Batch serves an op's batch mode: it receives objects until EOS/EOF, runs fn
// for each T and sends the returned reply — one reply per input, in input
// order. A failed input is fn returning an error object; the batch continues.
// A wrong-typed input is answered in-band with an error_message and the batch
// continues. The reply stream mirrors the input stream's terminator: an
// explicit EOS input is answered with a final EOS, while a stream ended by
// EOF is not — the caller is gone. Config functions (e.g. WithContext) pass
// through to Switch.
//
// With T = astral.Object the typed arm accepts every object and the
// wrong-type arm is unreachable; EOS still reaches its own arm because Switch
// consults exact-type handlers first.
func Batch[T astral.Object](ch *Channel, fn func(T) astral.Object, config ...any) error {
	var sawEOS bool
	args := []any{
		func(v T) error { return ch.Send(fn(v)) },
		func(obj astral.Object) error {
			return ch.Send(astral.Err(astral.NewErrUnexpectedObject(obj)))
		},
		MarkEOS(&sawEOS),
	}
	args = append(args, config...)

	// why: Switch fails on an input the channel cannot decode at all — an undecodable
	// JSON line, an unknown type tag on a binary channel, a corrupted frame. Returning
	// bare closed the channel with nothing on it, leaving the peer unable to tell a
	// rejected payload from a dropped transport. Report before closing. A failed report
	// is discarded: it means the channel is already broken, and the Switch error is the
	// one worth returning.
	if err := ch.Switch(args...); err != nil {
		_ = ch.Send(astral.Err(err))
		return err
	}
	if !sawEOS {
		return nil
	}
	return ch.Send(&astral.EOS{})
}
