package vfs

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ReadAction requests permission for Actor to read the VFS named by VFSID.
type ReadAction struct {
	auth.Action
	VFSID *astral.Identity
}

func (ReadAction) ObjectType() string { return "mod.vfs.read_action" }

func (a ReadAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ReadAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints allows the action when the bundle lists VFSID.
func (a ReadAction) ApplyConstraints(cs *astral.Bundle) bool { return lists(cs, a.VFSID) }

// WriteAction requests permission for Actor to change the VFS named by VFSID.
// It does not imply ReadAction.
type WriteAction struct {
	auth.Action
	VFSID *astral.Identity
}

func (WriteAction) ObjectType() string { return "mod.vfs.write_action" }

func (a WriteAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *WriteAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints allows the action when the bundle lists VFSID.
func (a WriteAction) ApplyConstraints(cs *astral.Bundle) bool { return lists(cs, a.VFSID) }

// ManageAction requests permission for Actor to administer VFSes.
type ManageAction struct {
	auth.Action
}

func (ManageAction) ObjectType() string { return "mod.vfs.manage_action" }

func (a ManageAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ManageAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. A manage
// permit has no constraints, and a constrained one grants nothing.
func (a ManageAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

// lists reports whether the bundle covers the VFS. The bundle covers exactly
// the identities it holds. A nil or empty bundle covers nothing, and a member
// that is not a non-zero identity makes the whole bundle cover nothing.
func lists(cs *astral.Bundle, vfs *astral.Identity) bool {
	if cs == nil {
		return false
	}

	members := cs.Objects()
	listed := false
	for _, o := range members {
		id, ok := o.(*astral.Identity)
		if !ok || id.IsZero() {
			return false
		}
		listed = listed || id.IsEqual(vfs)
	}

	return listed
}

func init() {
	astral.MustAdd(&ReadAction{})
	astral.MustAdd(&WriteAction{})
	astral.MustAdd(&ManageAction{})
}
