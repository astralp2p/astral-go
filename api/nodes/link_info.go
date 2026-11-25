package nodes

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/api/exonet"
	"github.com/astralp2p/astral-go/astral"
)

type LinkInfo struct {
	ID              astral.Nonce
	LocalIdentity   *astral.Identity
	RemoteIdentity  *astral.Identity
	LocalEndpoint   exonet.Endpoint
	RemoteEndpoint  exonet.Endpoint
	Outbound        astral.Bool
	Network         astral.String8
	HighPressure    astral.Bool
	BytesThroughput astral.Uint64
	Age             astral.Duration
	RTT             *astral.Duration // smoothed round trip of the keepalive pings; nil until the first pong
	RTTAge          *astral.Duration // time since the last pong; nil until the first pong
}

var _ astral.Object = &LinkInfo{}
var _ encoding.TextMarshaler = &LinkInfo{}
var _ json.Marshaler = &LinkInfo{}
var _ json.Unmarshaler = &LinkInfo{}

func (s LinkInfo) ObjectType() string {
	return "mod.nodes.link_info"
}

func (s LinkInfo) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&s).WriteTo(w)
}

func (s *LinkInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(s).ReadFrom(r)
}

func (s LinkInfo) MarshalText() (text []byte, err error) {
	var b = &bytes.Buffer{}

	var d = "<"
	if s.Outbound {
		d = ">"
	}

	_, err = fmt.Fprintf(b, "%v %v %v %v throughput=%v high=%v age=%v", s.ID, d, s.RemoteIdentity, s.RemoteEndpoint, s.BytesThroughput, bool(s.HighPressure), s.Age)
	if err == nil && s.RTT != nil {
		_, err = fmt.Fprintf(b, " rtt=%v", *s.RTT)
	}

	return b.Bytes(), err
}

func (e LinkInfo) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *LinkInfo) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

func init() {
	astral.MustAdd(&LinkInfo{})
}
