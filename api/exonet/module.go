package exonet

import (
	"github.com/astralp2p/astral-go/astral"
)

// Endpoint represents a dialable address on a network (such as an IP address with port number)
type Endpoint interface {
	astral.Object
	Network() string // network name
	Address() string // text representation of the address
	Pack() []byte    // binary representation of the address
}
