package vfs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// DirectoryType is the type name in the header of a directory, the canonical
// form of a folder. The ObjectID of a folder is the hash of its directory.
const DirectoryType = "mod.vfs.directory"

// entryMaxLen is the longest encoded entry: name length, name, ID, flags,
// modification time, creation time.
const entryMaxLen = 1 + MaxNameLen + 8 + 32 + 2 + 8 + 8

var (
	ErrDirectoryOrder = errors.New("directory entries out of order")
	ErrDirectoryFlags = errors.New("directory entry has an unknown flag bit")
	ErrDirectoryCount = errors.New("directory entry count mismatch")

	errDirectoryClosed = errors.New("directory closed")
)

// DirectoryWriter encodes the directory of a folder as a stream and hashes the
// bytes as it writes them. It never holds the directory as one value.
//
// The writer refuses an entry that is out of strict ascending byte order of
// Name, a repeated name, an invalid name and a non-zero unknown flag bit. The
// count of entries is fixed when the writer is made and must match the number
// of entries written.
type DirectoryWriter struct {
	out     *astral.WriteResolver
	count   uint32
	written uint32
	last    string
	buf     [entryMaxLen]byte
	failed  error
}

// NewDirectoryWriter writes the header of a directory of count entries and
// returns the writer. The directory bytes go to w. A nil w hashes only.
func NewDirectoryWriter(w io.Writer, count uint32) (*DirectoryWriter, error) {
	l := &DirectoryWriter{out: astral.NewWriteResolver(w), count: count}

	if err := l.writeHeader(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *DirectoryWriter) writeHeader() error {
	if _, err := (astral.Stamp{}).WriteTo(l.out); err != nil {
		return err
	}
	if _, err := astral.ObjectType(DirectoryType).WriteTo(l.out); err != nil {
		return err
	}

	return binary.Write(l.out, astral.ByteOrder, l.count)
}

// Write appends one entry. A refused entry leaves the writer unusable.
func (l *DirectoryWriter) Write(e *Entry) error {
	if l.failed != nil {
		return l.failed
	}

	l.failed = l.write(e)

	return l.failed
}

func (l *DirectoryWriter) write(e *Entry) error {
	if l.written >= l.count {
		return fmt.Errorf("%w: more than %d entries", ErrDirectoryCount, l.count)
	}
	if err := l.check(e); err != nil {
		return err
	}

	b := l.encode(e)
	if _, err := l.out.Write(b); err != nil {
		return err
	}

	l.last = string(e.Name)
	l.written++

	return nil
}

func (l *DirectoryWriter) check(e *Entry) error {
	if err := ValidateName(string(e.Name)); err != nil {
		return fmt.Errorf("%w: %q", err, string(e.Name))
	}
	if l.written > 0 && string(e.Name) <= l.last {
		return fmt.Errorf("%w: %q after %q", ErrDirectoryOrder, string(e.Name), l.last)
	}
	if e.Flags&^flagsKnown != 0 {
		return fmt.Errorf("%w: %#04x", ErrDirectoryFlags, uint16(e.Flags))
	}

	return nil
}

// encode lays out one entry in the scratch buffer.
// note: no type header and no presence byte, unlike the reflective encoder.
func (l *DirectoryWriter) encode(e *Entry) []byte {
	b := l.buf[:0]
	b = append(b, byte(len(e.Name)))
	b = append(b, e.Name...)
	b = binary.BigEndian.AppendUint64(b, e.ID.Size)
	b = append(b, e.ID.Hash[:]...)
	b = binary.BigEndian.AppendUint16(b, uint16(e.Flags))
	b = binary.BigEndian.AppendUint64(b, uint64(e.ModTime.Time().UnixNano()))
	b = binary.BigEndian.AppendUint64(b, uint64(e.CreationTime.Time().UnixNano()))

	return b
}

// Close ends the directory and returns the ObjectID of the bytes written: their
// count and their SHA-256. It refuses a directory with fewer entries than
// declared.
func (l *DirectoryWriter) Close() (*astral.ObjectID, error) {
	if l.failed != nil {
		return nil, l.failed
	}
	if l.written != l.count {
		l.failed = fmt.Errorf("%w: wrote %d of %d entries", ErrDirectoryCount, l.written, l.count)
		return nil, l.failed
	}
	l.failed = errDirectoryClosed

	return l.out.Resolve(), nil
}
