package astral

import "io"

// growChunk caps a single allocation step when reading a length-prefixed payload.
const growChunk = 64 << 10

// preallocElems caps how many elements a length-prefixed container reserves up
// front. Beyond it the container grows as elements arrive.
const preallocElems = 64

// readNBytes reads exactly want bytes from r, growing the buffer as bytes arrive
// rather than reserving want up front. It returns the bytes consumed even on
// failure, and returns io.ErrUnexpectedEOF for a short read.
//
// why: want is read off the wire. Reserving it before the first read let a
// four-byte header name a multi-gigabyte allocation — a 32-byte payload could
// reserve 4 GiB — so a peer could exhaust a node's memory without sending the
// bytes it claimed. Growing in bounded steps costs an allocation per chunk and
// bounds the reservation by what actually arrives.
func readNBytes(r io.Reader, want uint64) (buf []byte, n int64, err error) {
	if want == 0 {
		return nil, 0, nil
	}

	buf = make([]byte, 0, min(want, growChunk))

	for uint64(len(buf)) < want {
		grow := min(want-uint64(len(buf)), growChunk)
		start := len(buf)
		buf = append(buf, make([]byte, grow)...)

		var m int
		m, err = io.ReadFull(r, buf[start:])
		n += int64(m)

		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, n, err
		}
	}

	return buf, n, nil
}

// elemCap sizes the initial reservation of a container holding l elements.
func elemCap(l uint32) int {
	return int(min(uint64(l), preallocElems))
}
