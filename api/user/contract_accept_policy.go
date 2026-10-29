package user

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ContractAcceptRequest asks whether this node activates the fully signed node
// Contract that Caller submitted to user.accept_contract.
type ContractAcceptRequest struct {
	Caller   *astral.Identity
	Contract *auth.SignedContract
}

func (ContractAcceptRequest) ObjectType() string { return "mod.user.contract_accept_request" }

func (r ContractAcceptRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *ContractAcceptRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// ContractAcceptDecision is the contract accept policy's answer.
type ContractAcceptDecision struct {
	Allow astral.Bool
}

func (ContractAcceptDecision) ObjectType() string { return "mod.user.contract_accept_decision" }

func (d ContractAcceptDecision) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&d).WriteTo(w)
}

func (d *ContractAcceptDecision) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(d).ReadFrom(r)
}

func init() {
	astral.MustAdd(&ContractAcceptRequest{})
	astral.MustAdd(&ContractAcceptDecision{})
}
