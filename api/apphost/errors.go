package apphost

import "github.com/astralp2p/astral-go/astral"

// ErrRegistrationDeclined is the answer apphost.register sends once it has
// accepted the query and its policy refuses, or cannot decide.
var ErrRegistrationDeclined = astral.NewError("registration declined")

// ErrGrantDeclined is the answer apphost.request_grant sends once it has
// accepted the query and its policy refuses, or cannot decide.
var ErrGrantDeclined = astral.NewError("grant declined")
