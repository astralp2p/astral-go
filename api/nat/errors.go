package nat

import "github.com/astralp2p/astral-go/astral"

// ErrDisabled refuses a punch on a node that does not traverse NAT.
var ErrDisabled = astral.NewError("nat disabled")
