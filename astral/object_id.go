package astral

import (
	"bytes"
	"database/sql/driver"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const idPrefix = "data1"
const zBase32CharSet = "ybndrfg8ejkmcpqxot1uwisza345h769"

var zBase32Encoding = base32.NewEncoding(zBase32CharSet)

type ObjectID struct {
	Size uint64
	Hash [32]byte
}

func ParseID(s string) (id *ObjectID, err error) {
	// Check and trim the prefix
	if !strings.HasPrefix(s, idPrefix) {
		return nil, errors.New("invalid prefix")
	}
	s = strings.TrimPrefix(s, idPrefix)

	// Pad with missing leading zeros
	z := max(64-len(s), 0)
	padded := strings.Repeat(zBase32CharSet[0:1], z) + s

	var data [40]byte
	n, err := zBase32Encoding.Decode(data[:], []byte(padded))
	if err != nil {
		return nil, err
	}
	if n != 40 {
		return nil, errors.New("invalid data length")
	}

	id = &ObjectID{}
	id.Size = ByteOrder.Uint64(data[0:8])
	copy(id.Hash[:], data[8:40])

	return
}

// astral

func (ObjectID) ObjectType() string {
	return "object_id.sha256"
}

// WriteTo writes the id as Size||Hash, 40 bytes.
//
// why: this used to short-circuit on IsZero and write 40 zero bytes, which silently
// dropped a non-zero Size, so WriteTo followed by ReadFrom was not a round trip. The
// unconditional form emits the same 40 bytes for the true zero.
func (id *ObjectID) WriteTo(w io.Writer) (n int64, err error) {
	err = binary.Write(w, ByteOrder, id.Size)
	if err != nil {
		return
	}
	n += 8

	n2, err := w.Write(id.Hash[:])
	n += int64(n2)

	return
}

func (id *ObjectID) ReadFrom(r io.Reader) (n int64, err error) {
	err = binary.Read(r, ByteOrder, &id.Size)
	if err != nil {
		return
	}
	n += 8

	n2, err := io.ReadFull(r, id.Hash[:])
	n += int64(n2)

	return
}

// json

// MarshalJSON encodes the id as its data1 string.
//
// why: the zero id used to marshal to "", which UnmarshalJSON could not read back —
// ParseID rejects it. The zero id renders as the bare "data1" prefix, which round-trips,
// and which is what the text and SQL forms already emit.
func (id ObjectID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", id.String())), nil
}

func (id *ObjectID) UnmarshalJSON(b []byte) error {
	var s string
	var jsonDec = json.NewDecoder(bytes.NewReader(b))

	var err = jsonDec.Decode(&s)
	if err != nil {
		return err
	}

	// why: transitional — peers built against the previous encoder, and astral-py which
	// mirrors it, emit "" for the zero id. Accept it rather than erroring.
	if s == "" {
		*id = ObjectID{}
		return nil
	}

	parsed, err := ParseID(s)
	if err != nil {
		return err
	}

	*id = *parsed

	return nil
}

// text

func (id ObjectID) MarshalText() (text []byte, err error) {
	return []byte(id.String()), nil
}

func (id *ObjectID) UnmarshalText(text []byte) (err error) {
	parsed, err := ParseID(string(text))
	if err != nil {
		return err
	}
	*id = *parsed
	return
}

// sql

func (id ObjectID) Value() (driver.Value, error) {
	return id.String(), nil
}

func (id *ObjectID) Scan(src any) error {
	if src == nil {
		*id = ObjectID{}
		return nil
	}

	str, ok := src.(string)
	if !ok {
		return errors.New("typecast failed")
	}

	parsed, err := ParseID(str)
	if err != nil {
		return err
	}

	*id = *parsed

	return nil
}

// ...

func (id ObjectID) String() string {
	var b [40]byte
	ByteOrder.PutUint64(b[0:8], id.Size)
	copy(b[8:], id.Hash[0:32])
	enc := zBase32Encoding.EncodeToString(b[:])
	enc = strings.TrimLeft(enc, zBase32CharSet[0:1])
	return idPrefix + enc
}

// IsEqual compares both components.
//
// why: this used to short-circuit on IsZero, which ignored Size, so two ids differing
// only in Size compared equal to each other and to the true zero.
func (id *ObjectID) IsEqual(other *ObjectID) bool {
	if id.Size != other.Size {
		return false
	}

	return bytes.Compare(id.Hash[:], other.Hash[:]) == 0
}

// IsZero reports whether the id is the zero value — zero Size and all-zero Hash. A nil
// receiver counts as zero.
//
// why: Size is part of the id. Ignoring it made {Size:99, Hash:0} report as zero, which
// then dropped the Size on the binary wire, rendered as "" in JSON but as a data1 string
// in text and SQL, and compared equal to every other zero-hash id whatever its Size.
func (id *ObjectID) IsZero() bool {
	if id == nil {
		return true
	}

	if id.Size != 0 {
		return false
	}

	for _, b := range id.Hash {
		if b != 0 {
			return false
		}
	}
	return true
}

func init() {
	_ = Add(&ObjectID{})
}
