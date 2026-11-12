package vfs

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// FlagFolder is bit 0 of Entry.Flags. It is set for a folder and clear for a
// file. Bits 1 to 15 are zero in this version.
const FlagFolder astral.Uint16 = 1 << 0

// flagsKnown is the mask of the flag bits this version defines.
const flagsKnown = FlagFolder

// Entry is one name in a folder.
//
// ID is the ObjectID of the content for a file and the ObjectID of the
// listing for a folder. The size of a file is ID.Size. The folder bit never
// changes after the entry is created.
type Entry struct {
	Name    astral.String8
	ID      astral.ObjectID
	Flags   astral.Uint16
	ModTime astral.Time
}

func (Entry) ObjectType() string { return "mod.vfs.entry" }

func (e Entry) WriteTo(w io.Writer) (n int64, err error) { return astral.Objectify(&e).WriteTo(w) }

func (e *Entry) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e Entry) MarshalJSON() ([]byte, error) { return astral.Objectify(&e).MarshalJSON() }

func (e *Entry) UnmarshalJSON(b []byte) error { return astral.Objectify(e).UnmarshalJSON(b) }

// IsFolder reports whether the folder bit is set.
func (e Entry) IsFolder() bool { return e.Flags&FlagFolder != 0 }

func init() { astral.MustAdd(&Entry{}) }
