package dir

import "github.com/astralp2p/astral-go/astral"

// ErrUnknownIdentity refuses a name that no resolver turns into an identity.
var ErrUnknownIdentity = astral.NewError("unknown identity")
