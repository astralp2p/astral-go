package coldcard

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ScanAction requests permission for Actor to scan the node's attached Coldcard
// devices: enumerate them, request their derived public keys, and refresh the
// node's device map.
//
// It grants scanning alone. No signing or other device operation follows from
// it.
type ScanAction struct {
	auth.Action
}

func (ScanAction) ObjectType() string { return "mod.coldcard.scan_action" }

func (a ScanAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ScanAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a ScanAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&ScanAction{}) }
