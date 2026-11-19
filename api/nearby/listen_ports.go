package nearby

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// The tcp and kcp listen ports a nearby receiver assumes for a status without a port
// attachment.
const (
	DefaultTCPPort astral.Uint16 = 1791
	DefaultKCPPort astral.Uint16 = 1792
)

// ListenPorts is the sender's tcp and kcp listen ports. A Visible status carries it in
// plaintext, a Stealth status sealed as SealedPorts. A port of 0 means the sender does
// not listen on that network. A receiver dials the sender at the status's source IP on
// these ports.
type ListenPorts struct {
	TCP astral.Uint16
	KCP astral.Uint16
}

var _ astral.Object = &ListenPorts{}

func (ListenPorts) ObjectType() string { return "mod.nearby.listen_ports" }

func (p ListenPorts) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&p).WriteTo(w)
}

func (p *ListenPorts) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(p).ReadFrom(r)
}

// json

func (p ListenPorts) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&p).MarshalJSON()
}

func (p *ListenPorts) UnmarshalJSON(data []byte) error {
	return astral.Objectify(p).UnmarshalJSON(data)
}

// IsDefault reports whether p names the default ports, DefaultTCPPort and DefaultKCPPort.
func (p *ListenPorts) IsDefault() bool {
	return p.TCP == DefaultTCPPort && p.KCP == DefaultKCPPort
}

func init() {
	_ = astral.Add(&ListenPorts{})
}
