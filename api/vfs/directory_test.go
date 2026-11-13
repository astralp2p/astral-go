package vfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// vectorTime is the ModTime of every vector of VFS 6, section 5.
var vectorTime = astral.Time(time.Unix(0, 1700000000000000000))

// vectorCreationTime is the CreationTime of every vector, one hour before
// vectorTime.
var vectorCreationTime = astral.Time(time.Unix(0, 1699996400000000000))

func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}
	return b
}

func fileID(content string) astral.ObjectID {
	return astral.ObjectID{Size: uint64(len(content)), Hash: sha256.Sum256([]byte(content))}
}

type vector struct {
	name    string
	entries []Entry
	bytes   string
	sha256  string
}

// vectors are V1 to V4 of VFS 6, section 5, with the type name
// mod.vfs.directory and the creation time. The SHA-256 values are copied from
// the ticket "VFS: add a creation time to the entry".
func vectors() []vector {
	empty := astral.ObjectID{Size: 26}
	copy(empty.Hash[:], mustHex("5bda020d3782b67b601ab110d94f3a3d64396438e5b4c4a7c2cbf3d9ff565382"))

	return []vector{
		{
			name:   "V1 empty folder",
			bytes:  `41444330116d6f642e7666732e6469726563746f727900000000`,
			sha256: "5bda020d3782b67b601ab110d94f3a3d64396438e5b4c4a7c2cbf3d9ff565382",
		},
		{
			name: "V2 one file",
			entries: []Entry{
				{Name: "a.txt", ID: fileID("hello\n"), ModTime: vectorTime, CreationTime: vectorCreationTime},
			},
			bytes: `41444330116d6f642e7666732e6469726563746f727900000001
				05 612e747874
				0000000000000006
				5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03
				0000
				17979cfe362a0000
				179799b805716000`,
			sha256: "d7702ed2ff919b922b0b391123f977fe466455a7afcaba17f0528e66f0bc3982",
		},
		{
			name: "V3 one sub-folder",
			entries: []Entry{
				{Name: "sub", ID: empty, Flags: FlagFolder, ModTime: vectorTime, CreationTime: vectorCreationTime},
			},
			bytes: `41444330116d6f642e7666732e6469726563746f727900000001
				03 737562
				000000000000001a
				5bda020d3782b67b601ab110d94f3a3d64396438e5b4c4a7c2cbf3d9ff565382
				0001
				17979cfe362a0000
				179799b805716000`,
			sha256: "dab8a151d3e876ddb3c9cac823a0078a6a715698c82ae5b3b85b8d4abbb2805d",
		},
		{
			name: "V4 byte order of names",
			entries: []Entry{
				{Name: "z", ID: fileID("1"), ModTime: vectorTime, CreationTime: vectorCreationTime},
				{Name: "è", ID: fileID("2"), ModTime: vectorTime, CreationTime: vectorCreationTime},
				{Name: "é", ID: fileID("3"), ModTime: vectorTime, CreationTime: vectorCreationTime},
			},
			bytes: `41444330116d6f642e7666732e6469726563746f727900000003
				01 7a
				0000000000000001
				6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b
				0000 17979cfe362a0000 179799b805716000
				02 c3a8
				0000000000000001
				d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35
				0000 17979cfe362a0000 179799b805716000
				02 c3a9
				0000000000000001
				4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce
				0000 17979cfe362a0000 179799b805716000`,
			sha256: "d34da3f6c22c6532dc7737e39aca0b641256797c324fd7273435226586135de5",
		},
	}
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func encodeDirectory(entries []Entry, out *bytes.Buffer) (*astral.ObjectID, error) {
	l, err := NewDirectoryWriter(out, uint32(len(entries)))
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if err := l.Write(&entries[i]); err != nil {
			return nil, err
		}
	}
	return l.Close()
}

func TestDirectoryVectors(t *testing.T) {
	sizes := map[string]uint64{"V1": 26, "V2": 90, "V3": 88, "V4": 208}

	for _, v := range vectors() {
		t.Run(v.name, func(t *testing.T) {
			want := unhex(t, v.bytes)
			wantHash := mustHex(v.sha256)

			// the transcribed vector is self-consistent
			if sum := sha256.Sum256(want); !bytes.Equal(sum[:], wantHash) {
				t.Fatalf("spec bytes hash to %x, spec says %s", sum, v.sha256)
			}
			if size := sizes[v.name[:2]]; uint64(len(want)) != size {
				t.Fatalf("spec bytes are %d long, spec says %d", len(want), size)
			}

			var got bytes.Buffer
			id, err := encodeDirectory(v.entries, &got)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if !bytes.Equal(got.Bytes(), want) {
				t.Fatalf("bytes:\n got %x\nwant %x", got.Bytes(), want)
			}

			wantID := astral.ObjectID{Size: uint64(len(want))}
			copy(wantID.Hash[:], wantHash)
			if !id.IsEqual(&wantID) {
				t.Fatalf("object id: got %v, want %v", id, &wantID)
			}
		})
	}
}

// TestDirectoryHashOnly covers the writer made without an output: the ObjectID is
// the same.
func TestDirectoryHashOnly(t *testing.T) {
	for _, v := range vectors() {
		t.Run(v.name, func(t *testing.T) {
			l, err := NewDirectoryWriter(nil, uint32(len(v.entries)))
			if err != nil {
				t.Fatal(err)
			}
			for i := range v.entries {
				if err := l.Write(&v.entries[i]); err != nil {
					t.Fatal(err)
				}
			}
			id, err := l.Close()
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(id.Hash[:]) != v.sha256 {
				t.Fatalf("hash: got %x, want %s", id.Hash, v.sha256)
			}
		})
	}
}

func TestDirectoryRefusals(t *testing.T) {
	a := Entry{Name: "a", ID: fileID("a"), ModTime: vectorTime, CreationTime: vectorCreationTime}
	b := Entry{Name: "b", ID: fileID("b"), ModTime: vectorTime, CreationTime: vectorCreationTime}

	for _, tc := range []struct {
		name    string
		entries []Entry
		count   uint32
		wantErr error
	}{
		{"out of order", []Entry{b, a}, 2, ErrDirectoryOrder},
		{"repeated name", []Entry{a, a}, 2, ErrDirectoryOrder},
		{"invalid name empty", []Entry{{Name: "", ID: a.ID}}, 1, ErrInvalidPath},
		{"invalid name slash", []Entry{{Name: "x/y", ID: a.ID}}, 1, ErrInvalidPath},
		{"invalid name dot", []Entry{{Name: ".", ID: a.ID}}, 1, ErrInvalidPath},
		{"unknown flag bit", []Entry{{Name: "a", ID: a.ID, Flags: 0x0002}}, 1, ErrDirectoryFlags},
		{"unknown flag bit with folder", []Entry{{Name: "a", ID: a.ID, Flags: 0x8001}}, 1, ErrDirectoryFlags},
		{"too many entries", []Entry{a, b}, 1, ErrDirectoryCount},
		{"too few entries", []Entry{a}, 2, ErrDirectoryCount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, err := NewDirectoryWriter(nil, tc.count)
			if err != nil {
				t.Fatal(err)
			}
			for i := range tc.entries {
				if err = l.Write(&tc.entries[i]); err != nil {
					break
				}
			}
			if err == nil {
				_, err = l.Close()
			}
			if !errors.Is(err, tc.wantErr) && !sameError(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// sameError matches an astral error by its text, as the SDK does.
func sameError(err, target error) bool {
	var t astral.Error
	return errors.As(target, &t) && errors.Is(err, t)
}

// TestDirectoryFailureIsSticky covers a writer that refused an entry: it accepts
// nothing more and does not close.
func TestDirectoryFailureIsSticky(t *testing.T) {
	l, err := NewDirectoryWriter(nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	bad := Entry{Name: "a", ID: fileID("a"), Flags: 0x0002}
	if l.Write(&bad) == nil {
		t.Fatal("unknown flag accepted")
	}
	good := Entry{Name: "b", ID: fileID("b")}
	if l.Write(&good) == nil {
		t.Fatal("write after refusal accepted")
	}
	if _, err := l.Close(); err == nil {
		t.Fatal("close after refusal succeeded")
	}
}

func TestDirectoryLongestName(t *testing.T) {
	name := strings.Repeat("n", MaxNameLen)
	var out bytes.Buffer
	_, err := encodeDirectory([]Entry{{Name: astral.String8(name), ID: fileID("x"), ModTime: vectorTime, CreationTime: vectorCreationTime}}, &out)
	if err != nil {
		t.Fatalf("255-byte name refused: %v", err)
	}
	if out.Len() != 26+entryMaxLen {
		t.Fatalf("size: got %d, want %d", out.Len(), 26+entryMaxLen)
	}
}
