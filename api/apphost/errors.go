package apphost

import "github.com/astralp2p/astral-go/astral"

// ErrRegistrationDeclined is the answer apphost.register sends once it has
// accepted the query and its policy refuses, or cannot decide.
var ErrRegistrationDeclined = astral.NewError("registration declined")

// ErrGrantDeclined is the answer apphost.request_grant sends once it has
// accepted the query and its policy refuses, or cannot decide.
var ErrGrantDeclined = astral.NewError("grant declined")

// ErrNotFresh is the answer apphost.register_manager sends on a node that is
// not fresh: a manager is appointed, or the node is or was owned.
var ErrNotFresh = astral.NewError("node is not fresh")
