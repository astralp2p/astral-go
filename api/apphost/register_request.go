package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// RegisterRequest is the registration an app sends to apphost.register. The
// node passes it to the app register policy as an AppRegisterRequest, which
// adds the caller's web origin.
//
// Name is the name the app gives itself; the node does not verify it.
// GrantPermits are the permits the app asks to hold as node-local grants;
// ContractPermits are the permits it asks to hold in a signed node→app
// contract. Both carry their constraints.
type RegisterRequest struct {
	Name            astral.String8
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
}

func (RegisterRequest) ObjectType() string { return "mod.apphost.register_request" }

func (r RegisterRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *RegisterRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r RegisterRequest) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&r).MarshalJSON()
}

func (r *RegisterRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

func init() {
	astral.MustAdd(&RegisterRequest{})
}
