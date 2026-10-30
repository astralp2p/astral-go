package auth

import (
	"io"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// RegisterSignerAction requests permission for Actor to claim Key on a
// crypto.register_signer binding.
//
// It grants no signature and no right to act as Key's identity. A holder sees
// what is sent for signing under the keys it claims, and may refuse it; the node
// verifies every signature a holder returns.
//
// why in this package: api/auth imports api/crypto, so an action embedding
// Action cannot live in api/crypto.
type RegisterSignerAction struct {
	Action
	Key *crypto.PublicKey
}

func (RegisterSignerAction) ObjectType() string { return "mod.auth.register_signer_action" }

func (a RegisterSignerAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *RegisterSignerAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a RegisterSignerAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&RegisterSignerAction{}) }
