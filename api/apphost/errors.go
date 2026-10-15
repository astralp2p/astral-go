package apphost

import "github.com/astralp2p/astral-go/astral"

// ErrRegistrationDeclined is the answer apphost.register sends once it has
// accepted the query and its policy refuses, or cannot decide.
var ErrRegistrationDeclined = astral.NewError("registration declined")
