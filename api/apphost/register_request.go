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
//
// Vfs asks the node to create a VFS for the app. Evaluator asks the node to
// name the app the evaluator of that VFS, at the endpoint
// astral://<app identity>:vfs.authorize. A nil field means true, so a request
// that omits both gets a VFS that the app evaluates. Evaluator has no effect
// when Vfs is false.
type RegisterRequest struct {
	Name            astral.String8
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
	Vfs             *astral.Bool
	Evaluator       *astral.Bool
}

// RequestsVFS reports whether the request asks for a VFS. A nil Vfs asks.
func (r *RegisterRequest) RequestsVFS() bool { return r.Vfs == nil || bool(*r.Vfs) }

// RequestsEvaluator reports whether the request asks for the app to be the
// evaluator of its VFS. A nil Evaluator asks. A request without a VFS asks
// for no evaluator.
func (r *RegisterRequest) RequestsEvaluator() bool {
	return r.RequestsVFS() && (r.Evaluator == nil || bool(*r.Evaluator))
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
