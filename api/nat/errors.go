package nat

import "github.com/astralp2p/astral-go/astral"

// ErrDisabled refuses a punch on a node that does not traverse NAT.
var ErrDisabled = astral.NewError("nat disabled")

// ErrNoReflection answers a nat.reflect_udp query whose reflector received no
// datagram carrying its token within ReflectTimeout.
var ErrNoReflection = astral.NewError("no reflected datagram")
