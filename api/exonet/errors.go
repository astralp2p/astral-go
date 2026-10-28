package exonet

import "github.com/astralp2p/astral-go/astral"

// ErrInvalidEndpoint refuses an endpoint that is not a parsable "network:address".
var ErrInvalidEndpoint = astral.NewError("invalid endpoint")

// ErrUnsupportedNetwork refuses an endpoint on a network no parser is registered for.
var ErrUnsupportedNetwork = astral.NewError("unsupported network")
