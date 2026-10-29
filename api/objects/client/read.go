package objects

import (
	"errors"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Read returns a stream of the object's bytes from offset; caller must Close it.
// It reads objects.read in its framed mode, so a refusal or a missing object
// returns its error, such as objects.ErrNotFound or auth.ErrNotPermitted.
func (client *Client) Read(ctx *astral.Context, objectID *astral.ObjectID, offset, limit int64) (io.ReadCloser, error) {
	ch, err := client.queryCh(ctx, objects.MethodRead, query.Args{
		"id":     objectID,
		"offset": offset,
		"limit":  limit,
		"zone":   "dvn",
		"out":    channel.Binary,
	})
	if err != nil {
		return nil, err
	}

	// why the first object is received here: a refusal is the only object its
	// stream carries, so the caller learns it from Read and not from a later read.
	r := &chunkReader{ch: ch}
	r.next()
	if r.err != nil && !errors.Is(r.err, io.EOF) {
		ch.Close()
		return nil, r.err
	}
	return r, nil
}

func Read(ctx *astral.Context, objectID *astral.ObjectID, offset, limit int64) (io.ReadCloser, error) {
	return Default().Read(ctx, objectID, offset, limit)
}

// chunkReader reads the bytes32 chunks of a framed objects.read as one stream.
type chunkReader struct {
	ch  *channel.Channel
	buf []byte
	err error
}

func (r *chunkReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		r.next()
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func (r *chunkReader) Close() error {
	return r.ch.Close()
}

// next receives one object: a chunk fills buf, eos ends the stream with io.EOF,
// and an error_message ends it with that error.
func (r *chunkReader) next() {
	obj, err := r.ch.Receive()
	if err != nil {
		r.err = err
		return
	}

	switch o := obj.(type) {
	case *astral.Bytes32:
		r.buf = *o
	case *astral.EOS:
		r.err = io.EOF
	case *astral.ErrorMessage:
		r.err = o
	default:
		r.err = fmt.Errorf("objects.read answered %v", obj.ObjectType())
	}
}
