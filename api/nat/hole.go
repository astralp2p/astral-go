package nat

import (
	"encoding/json"
	"io"
	"net"

	"github.com/astralp2p/astral-go/astral"
)

// Hole represents a pair of connected endpoints resulting from a successful nat.punch operation.
//
// An endpoint is the external address the other side observed; a bind port is
// the local UDP port its owner binds. A NAT that translates ports makes them differ.
type Hole struct {
	Nonce           astral.Nonce
	ActiveIdentity  *astral.Identity
	ActiveEndpoint  Endpoint
	ActiveBindPort  astral.Uint16
	PassiveIdentity *astral.Identity
	PassiveEndpoint Endpoint
	PassiveBindPort astral.Uint16
	CreatedAt       astral.Time
}

// ObjectType implements astral.Object.
func (h Hole) ObjectType() string {
	return "nat.hole"
}

func (h Hole) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&h).WriteTo(w)
}

func (h *Hole) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(h).ReadFrom(r)
}

// MarshalJSON encodes Hole into JSON.
func (h Hole) MarshalJSON() ([]byte, error) {
	type alias Hole
	return json.Marshal(alias(h))
}

// UnmarshalJSON decodes Hole from JSON.
func (h *Hole) UnmarshalJSON(bytes []byte) error {
	type alias Hole
	var a alias
	if err := json.Unmarshal(bytes, &a); err != nil {
		return err
	}
	*h = Hole(a)
	return nil
}

// RemoteIdentity returns the identity of the remote peer.
func (h *Hole) RemoteIdentity(self *astral.Identity) (*astral.Identity, bool) {
	switch {
	case h.ActiveIdentity != nil && h.ActiveIdentity.IsEqual(self):
		return h.PassiveIdentity, true
	case h.PassiveIdentity != nil && h.PassiveIdentity.IsEqual(self):
		return h.ActiveIdentity, true
	default:
		return nil, false
	}
}

// GetLocalAddr returns the local UDP address for this hole.
func (h *Hole) GetLocalAddr(self *astral.Identity) *net.UDPAddr {
	if h.ActiveIdentity.IsEqual(self) {
		return h.ActiveEndpoint.UDPAddr()
	}
	return h.PassiveEndpoint.UDPAddr()
}

// GetRemoteAddr returns the remote UDP address for this hole.
func (h *Hole) GetRemoteAddr(self *astral.Identity) *net.UDPAddr {
	if h.ActiveIdentity.IsEqual(self) {
		return h.PassiveEndpoint.UDPAddr()
	}
	return h.ActiveEndpoint.UDPAddr()
}

// GetLocalBindPort returns the local UDP port this node binds for the hole.
func (h *Hole) GetLocalBindPort(self *astral.Identity) int {
	if h.ActiveIdentity.IsEqual(self) {
		return int(h.ActiveBindPort)
	}
	return int(h.PassiveBindPort)
}

// GetRemoteBindPort returns the local UDP port the remote peer binds for the hole.
func (h *Hole) GetRemoteBindPort(self *astral.Identity) int {
	if h.ActiveIdentity.IsEqual(self) {
		return int(h.PassiveBindPort)
	}
	return int(h.ActiveBindPort)
}

func (h *Hole) MatchesPeer(peer *astral.Identity) bool {
	return h.ActiveIdentity.IsEqual(peer) || h.PassiveIdentity.IsEqual(peer)
}

func init() {
	astral.MustAdd(&Hole{})
}
