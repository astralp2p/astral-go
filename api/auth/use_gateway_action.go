package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// UseGatewayAction requests permission for Actor to use this node's public
// gateway service: register as a gateway-reachable node, reserve a connection
// to one, and forward a routed connection through the node.
//
// Gateway use is service consumption, not administration. It grants nothing
// AdminNetworkAction governs.
type UseGatewayAction struct {
	Action
}

func (UseGatewayAction) ObjectType() string { return "mod.auth.use_gateway_action" }

func (a UseGatewayAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *UseGatewayAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a UseGatewayAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&UseGatewayAction{}) }
