package astral

import "io"

// writePayload writes the body of a length-prefixed value, and writes nothing at all
// when the body is empty.
//
// why: w.Write(nil) is not universally a no-op. On io.Pipe it parks until a reader takes
// the bytes, and the peer never asks for them — ReadFrom returns on a zero length
// without issuing a Read, so there is no matching call to unblock it. In-process routing
// is io.Pipe in both directions, and Ack, EOS and Nil all embed EmptyObject and
// serialise to zero bytes, so the values that hit this are the framework's own stream
// terminators. Over TCP the same call is a harmless no-op, which is why it stayed
// invisible in normal use.
//
// The count is unchanged: an empty payload contributes nothing beyond its length prefix,
// which the caller has already counted.
func writePayload(w io.Writer, b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, nil
	}

	n, err := w.Write(b)

	return int64(n), err
}
