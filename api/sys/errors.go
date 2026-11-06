package sys

import "github.com/astralp2p/astral-go/astral"

// ErrNotFresh is the answer sys.register_manager sends on a node that is not
// fresh: a manager is appointed, or the node is or was owned.
var ErrNotFresh = astral.NewError("node is not fresh")
