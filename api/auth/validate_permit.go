package auth

import (
	"errors"
	"fmt"

	"github.com/astralp2p/astral-go/astral"
)

var errMissingAction = errors.New("missing action")
var errUnknownAction = errors.New("unknown action")
var errRequiresConstraints = errors.New("requires constraints")

// ValidatePermit reports a permit the node must refuse before asking any
// policy or writing anything. The error wraps ErrInvalidPermit and names the action.
//
// The rules apply in order:
//   - a nil permit or an empty Action is refused;
//   - an Action with no registered blueprint is refused;
//   - a Constrainable action whose ApplyConstraints(nil) is false is refused
//     when the permit carries no constraints;
//   - a ConstraintValidator action is refused when the permit carries
//     constraints and ValidateConstraints returns an error.
//
// Every other permit passes. A non-constrainable action with constraints passes as sent.
func ValidatePermit(p *Permit) error {
	if p == nil || p.Action == "" {
		return fmt.Errorf("%w: %v", ErrInvalidPermit, errMissingAction)
	}
	action := string(p.Action)

	prototype := astral.New(action)
	if prototype == nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidPermit, action, errUnknownAction)
	}

	constrained := p.Constraints != nil && len(p.Constraints.Objects()) > 0

	if c, ok := prototype.(Constrainable); ok && !constrained && !c.ApplyConstraints(nil) {
		return fmt.Errorf("%w: %s: %v", ErrInvalidPermit, action, errRequiresConstraints)
	}

	if v, ok := prototype.(ConstraintValidator); ok && constrained {
		if err := v.ValidateConstraints(p.Constraints); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrInvalidPermit, action, err)
		}
	}

	return nil
}
