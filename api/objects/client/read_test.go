package objects

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// framed encodes objs as a framed objects.read answers them and returns a
// chunkReader over them.
func framed(t *testing.T, objs ...astral.Object) *chunkReader {
	t.Helper()

	var in bytes.Buffer
	send := channel.NewSender(&in)
	for _, obj := range objs {
		if err := send.Send(obj); err != nil {
			t.Fatalf("encode %v: %v", obj.ObjectType(), err)
		}
	}
	return &chunkReader{ch: channel.New(channel.Join(&in, io.Discard))}
}

func chunk(s string) *astral.Bytes32 {
	b := astral.Bytes32(s)
	return &b
}

// The chunks of a framed read are one stream, ended by eos.
func TestChunkReader_JoinsTheChunks(t *testing.T) {
	r := framed(t, chunk("hello "), chunk("world"), &astral.EOS{})

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("read %q; want %q", got, "hello world")
	}
}

// An error_message after some chunks fails the read with that error.
func TestChunkReader_EndsWithTheAnsweredError(t *testing.T) {
	r := framed(t, chunk("partial"), objects.ErrNotFound)

	got, err := io.ReadAll(r)
	if string(got) != "partial" {
		t.Fatalf("read %q before the error; want %q", got, "partial")
	}
	if !errors.Is(err, objects.ErrNotFound) {
		t.Fatalf("error %v; want %v", err, objects.ErrNotFound)
	}
}

// An empty object is eos alone: the read ends with no bytes and no error.
func TestChunkReader_EmptyObject(t *testing.T) {
	r := framed(t, &astral.EOS{})

	got, err := io.ReadAll(r)
	if err != nil || len(got) != 0 {
		t.Fatalf("read %q, %v; want nothing and no error", got, err)
	}
}
