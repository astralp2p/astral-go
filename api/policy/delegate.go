// Package policy holds what a node and the app it delegates an admission
// decision to share: the delegate, its bound, and the one way to ask it.
package policy

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// MaxTimeout bounds one admission question. The wait for a delegate ends at
// the earliest of its answer, the end of the asking context, and this bound.
const MaxTimeout = 10 * time.Minute

// Delegate names the op that answers an admission question instead of the
// node, and how long the node waits for the answer. A zero Timeout waits
// MaxTimeout; a Timeout above it fails Validate.
type Delegate struct {
	Identity  *astral.Identity
	Operation astral.String8
	Timeout   astral.Duration
}

func (Delegate) ObjectType() string { return "policy.delegate" }

func (d Delegate) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&d).WriteTo(w)
}

func (d *Delegate) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(d).ReadFrom(r)
}

// IsZero reports whether the delegate names no identity, which selects the
// module's bundled policy.
func (d *Delegate) IsZero() bool {
	return d == nil || d.Identity == nil || d.Identity.IsZero()
}

// Validate reports a delegate that names no identity or no operation, or
// waits longer than MaxTimeout.
func (d *Delegate) Validate() error {
	switch {
	case d.IsZero():
		return errors.New("policy delegate: no identity named")
	case d.Operation == "":
		return errors.New("policy delegate: no operation named")
	case time.Duration(d.Timeout) > MaxTimeout:
		return fmt.Errorf("policy delegate: timeout above %v", MaxTimeout)
	}
	return nil
}

// timeout is the bound of one question: Timeout when set, else MaxTimeout.
func (d *Delegate) timeout() time.Duration {
	if d.Timeout <= 0 {
		return MaxTimeout
	}
	return time.Duration(d.Timeout)
}

func init() {
	astral.MustAdd(&Delegate{})
}
