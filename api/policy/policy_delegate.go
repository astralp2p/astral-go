// Package policy holds the types shared by node policies whose decision a
// module can delegate to another identity.
package policy

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// PolicyDelegate names where a delegated policy sends its question: the
// identity that decides and the operation it serves. Both are configured
// together; neither has a default.
type PolicyDelegate struct {
	Identity  *astral.Identity
	Operation astral.String8
}

func (PolicyDelegate) ObjectType() string { return "policy.delegate" }

func (d PolicyDelegate) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&d).WriteTo(w)
}

func (d *PolicyDelegate) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(d).ReadFrom(r)
}

// IsZero reports whether the delegate names no identity, which selects the
// module's default policy.
func (d *PolicyDelegate) IsZero() bool {
	return d == nil || d.Identity == nil || d.Identity.IsZero()
}

func init() {
	astral.MustAdd(&PolicyDelegate{})
}
