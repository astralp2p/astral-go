package vfs

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Info is the facts about one VFS.
//
// Usage counts the bytes of file content, each distinct ObjectID once. Quota
// and EntryLimit are limits; 0 means no limit. Entries does not count the
// root.
type Info struct {
	VFS        *astral.Identity
	Root       astral.ObjectID
	Usage      astral.Uint64
	Quota      astral.Uint64
	Entries    astral.Uint64
	EntryLimit astral.Uint64
}

func (Info) ObjectType() string { return "mod.vfs.info" }

func (i Info) WriteTo(w io.Writer) (n int64, err error) { return astral.Objectify(&i).WriteTo(w) }

func (i *Info) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(i).ReadFrom(r)
}

func (i Info) MarshalJSON() ([]byte, error) { return astral.Objectify(&i).MarshalJSON() }

func (i *Info) UnmarshalJSON(b []byte) error { return astral.Objectify(i).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Info{}) }
