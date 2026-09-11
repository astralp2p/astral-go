package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// AdminNetworkAction requests permission for Actor to administer the node's
// network: read its links, sessions, endpoints, addresses and NAT state; make
// it dial, punch, broadcast and consume traversal holes; and open, close and
// remap its links and listeners.
//
// One action covers the reads, the work and the control across the nodes, ip,
// nat, kcp, tcp, nearby and services modules. The grouping is deliberate: a
// holder of one tier holds the others.
//
// Using this node's public gateway is UseGatewayAction, not this action.
type AdminNetworkAction struct {
	Action
}

func (AdminNetworkAction) ObjectType() string { return "mod.auth.admin_network_action" }

func (a AdminNetworkAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *AdminNetworkAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a AdminNetworkAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&AdminNetworkAction{}) }
