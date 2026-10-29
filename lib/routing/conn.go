package routing

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Conn is a basic implementation of an astral.Conn over a Reader and a WriteCloser.
type Conn struct {
	localID  *astral.Identity
	remoteID *astral.Identity
	io.WriteCloser
	io.Reader
	outbound bool
}

var _ astral.Conn = &Conn{}

func NewConn(localID *astral.Identity, remoteID *astral.Identity, w io.WriteCloser, r io.Reader, outbound bool) astral.Conn {
	return &Conn{
		localID:     localID,
		remoteID:    remoteID,
		WriteCloser: w,
		Reader:      r,
		outbound:    outbound,
	}
}

// Read closes the connection on any read error to ensure cleanup propagates.
func (s *Conn) Read(p []byte) (n int, err error) {
	n, err = s.Reader.Read(p)
	if err != nil {
		s.Close()
	}
	return n, err
}

// Close closes the write half, and the read half when it can be closed.
//
// why: AcceptRaw's read half is a pipe the caller writes into. An op that closes
// its connection without reading, such as one answering a refusal, would
// otherwise leave the caller's next write parked forever.
func (s *Conn) Close() error {
	if r, ok := s.Reader.(io.Closer); ok {
		r.Close()
	}
	return s.WriteCloser.Close()
}

func (s *Conn) Outbound() bool {
	return s.outbound
}

func (s *Conn) RemoteIdentity() *astral.Identity {
	return s.remoteID
}

func (s *Conn) LocalIdentity() *astral.Identity {
	return s.localID
}
