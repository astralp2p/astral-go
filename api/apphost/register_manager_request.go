package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// The policy names a PolicyBinding takes. Each names the admission policy of
// one module op: app_register and grant_request decide apphost.register and
// apphost.request_grant; swarm_join, swarm_invite and contract_accept decide
// user.request_membership, user.accept_membership and user.accept_contract.
const (
	PolicyAppRegister    = "app_register"
	PolicyGrantRequest   = "grant_request"
	PolicySwarmJoin      = "swarm_join"
	PolicySwarmInvite    = "swarm_invite"
	PolicyContractAccept = "contract_accept"
)

// PolicyNames lists every policy name a PolicyBinding takes.
var PolicyNames = []string{
	PolicyAppRegister,
	PolicyGrantRequest,
	PolicySwarmJoin,
	PolicySwarmInvite,
	PolicyContractAccept,
}

// RegisterManagerRequest is the registration an app sends to
// apphost.register_manager to become the node's manager. Name, GrantPermits
// and ContractPermits mean what they mean in RegisterRequest. Policies names
// the policies the manager answers: the node stores each as a delegate whose
// identity is the new manager.
type RegisterManagerRequest struct {
	Name            astral.String8
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
	Policies        []*PolicyBinding
}

func (RegisterManagerRequest) ObjectType() string { return "mod.apphost.register_manager_request" }

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

// PolicyBinding names the op of the manager that answers one policy. Policy
// is one of PolicyNames. Operation is the op the node asks. Timeout bounds
// one question as policy.Delegate.Timeout does.
type PolicyBinding struct {
	Policy    astral.String8
	Operation astral.String8
	Timeout   astral.Duration
}

func (PolicyBinding) ObjectType() string { return "mod.apphost.policy_binding" }

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
