package user

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// OpUpdate is one entry of the asset operation log streamed by OpSyncAssets.
// An entry either adds an object to the user's asset list or, when Removed is
// set, tombstones it; the nonce identifies the entry so a replayed log
// converges on the same list.
type OpUpdate struct {
	Nonce    astral.Nonce
	ObjectID *astral.ObjectID
	Removed  astral.Bool
}

var _ astral.Object = &OpUpdate{}

func (OpUpdate) ObjectType() string { return "mod.user.op_update" }

func (u OpUpdate) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&u).WriteTo(w)
}

func (u *OpUpdate) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(u).ReadFrom(r)
}

func init() {
	astral.MustAdd(&OpUpdate{})
}
