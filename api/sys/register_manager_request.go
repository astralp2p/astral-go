package sys

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// RegisterManagerRequest is the registration an app sends to
// sys.register_manager to become the node's manager. Name, GrantPermits and
// ContractPermits mean what they mean in apphost.RegisterRequest. Policies
// stays on the wire only: the node refuses a request that names any.
type RegisterManagerRequest struct {
	Name            astral.String8
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
	Policies        []*PolicyBinding
}

func (RegisterManagerRequest) ObjectType() string { return "mod.sys.register_manager_request" }

func (r RegisterManagerRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *RegisterManagerRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r RegisterManagerRequest) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&r).MarshalJSON()
}

func (r *RegisterManagerRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

// PolicyBinding is the wire form of one entry of
// RegisterManagerRequest.Policies. The node refuses every binding.
type PolicyBinding struct {
	Policy    astral.String8
	Operation astral.String8
	Timeout   astral.Duration
}

func (PolicyBinding) ObjectType() string { return "mod.sys.policy_binding" }

func (b PolicyBinding) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&b).WriteTo(w)
}

func (b *PolicyBinding) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(b).ReadFrom(r)
}

func (b PolicyBinding) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&b).MarshalJSON()
}

func (b *PolicyBinding) UnmarshalJSON(data []byte) error {
	return astral.Objectify(b).UnmarshalJSON(data)
}

func init() {
	astral.MustAdd(&RegisterManagerRequest{})
	astral.MustAdd(&PolicyBinding{})
}
