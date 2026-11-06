package user

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// The node states user.info reports in Info.State.
const (
	// StateFresh is a node with no manager and no ownership history. It admits
	// the appointment of a manager.
	StateFresh = "fresh"
	// StateUnclaimed is a node without an active user contract whose
	// appointment is closed: it has a manager, or it was owned before.
	StateUnclaimed = "unclaimed"
	// StateClaimed is a node with an active user contract.
	StateClaimed = "claimed"
)

// Info contains information about user's node configuration.
//
// State is one of StateFresh, StateUnclaimed and StateClaimed. Manager is the
// App identity sys.register_manager appointed, nil when none is appointed.
// NodeAlias, UserAlias, ContractID and Contract are empty unless State is
// StateClaimed.
type Info struct {
	NodeAlias  astral.String8
	UserAlias  astral.String8
	ContractID *astral.ObjectID

	Contract *auth.SignedContract

	State   astral.String8
	Manager *astral.Identity
}

func (i Info) ObjectType() string { return "mod.user.info" }

func (i Info) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&i).WriteTo(w)
}

func (i *Info) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(i).ReadFrom(r)
}

func init() {
	astral.MustAdd(&Info{})
}
