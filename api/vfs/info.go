package vfs

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Info is the facts about one VFS.
//
// Usage counts the bytes of file content, each distinct ObjectID once. Quota
// and InodeLimit are limits; 0 means no limit. Inodes counts the entries of
// the VFS, files and folders; it does not count the root. Evaluator is the
// endpoint the node asks about a caller its own rules refuse, of the form
// astral://<identity-or-alias>:vfs.authorize; empty means no evaluator.
type Info struct {
	VFS        *astral.Identity
	Root       astral.ObjectID
	Usage      astral.Uint64
	Quota      astral.Uint64
	Inodes     astral.Uint64
	InodeLimit astral.Uint64
	Evaluator  astral.String8
}

func (Info) ObjectType() string { return "mod.vfs.info" }

func (i Info) WriteTo(w io.Writer) (n int64, err error) { return astral.Objectify(&i).WriteTo(w) }

func (i *Info) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(i).ReadFrom(r)
}

func (i Info) MarshalJSON() ([]byte, error) { return astral.Objectify(&i).MarshalJSON() }

func (i *Info) UnmarshalJSON(b []byte) error { return astral.Objectify(i).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Info{}) }
