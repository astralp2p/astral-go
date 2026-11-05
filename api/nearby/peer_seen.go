package nearby

import (
	"io"

	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/astral"
)

// The PeerSeen.Via values: the field of the status that identified the peer.
const (
	ViaHint    = "hint"
	ViaProfile = "profile"
)

// PeerSeen is the events.Event data the nearby module emits locally after a status
// passes verification. UserID is nil when the peer's user is unknown; a reader tests it
// with IsZero, which also covers a zero identity. Via is ViaHint or ViaProfile.
// SourceIP is the source address of the datagram.
type PeerSeen struct {
	NodeID   *astral.Identity
	UserID   *astral.Identity
	Via      astral.String8
	SourceIP ip.IP
}

var _ astral.Object = &PeerSeen{}

func (PeerSeen) ObjectType() string { return "mod.nearby.peer_seen" }

func (e PeerSeen) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *PeerSeen) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(e).ReadFrom(r)
}

// json

func (e PeerSeen) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *PeerSeen) UnmarshalJSON(data []byte) error {
	return astral.Objectify(e).UnmarshalJSON(data)
}

func init() {
	_ = astral.Add(&PeerSeen{})
}
