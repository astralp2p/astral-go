package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Change tells the node, on an advertisement binding, that the offerings of
// some callers may have changed. With All false, Callers names them and must be
// non-empty; with All true, it covers every caller following a service of the
// binding.
type Change struct {
	All     astral.Bool
	Callers []*astral.Identity
}

func (Change) ObjectType() string { return "services.change" }

func (c Change) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&c).WriteTo(w)
}

func (c *Change) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(c).ReadFrom(r)
}

// Valid reports whether the node accepts c. A Change selecting no caller fails
// the binding.
func (c Change) Valid() bool {
	return bool(c.All) || len(c.Callers) > 0
}

func init() { astral.MustAdd(&Change{}) }
