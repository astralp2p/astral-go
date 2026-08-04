package astral

import (
	"encoding/binary"
	"io"
	"strconv"
)

// fixedWidth is the set of Go builtins the scalar astral types wrap.
type fixedWidth interface {
	bool | int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

// writeFixed writes v in ByteOrder and reports the width binary.Write emitted.
//
// why: the byte count used to be a hand-written literal beside the value, a second
// and unchecked source of truth. Six of the ten scalars disagreed with the bytes they
// moved — Uint16 through Uint64 and Int16 through Int64 all reported 1, and Float64
// reported 4 for 8. Deriving the width from the value removes the place to get it
// wrong. Callers pass the builtin rather than the named type so binary.Write keeps
// its fast path instead of falling back to reflection.
func writeFixed[T fixedWidth](w io.Writer, v T) (int64, error) {
	if err := binary.Write(w, ByteOrder, v); err != nil {
		return 0, err
	}
	return int64(binary.Size(v)), nil
}

// readFixed reads one fixed-width value in ByteOrder and reports the width consumed.
func readFixed[T fixedWidth](r io.Reader, v *T) (int64, error) {
	if err := binary.Read(r, ByteOrder, v); err != nil {
		return 0, err
	}
	return int64(binary.Size(*v)), nil
}

// parseIntText parses text as a base-10 signed integer of exactly T's wire width.
//
// why: Int16, Int32 and Int64 all parsed at bit size 8, copied from Int8 where 8 is
// correct. Any value outside int8 range failed, so these types could not re-read the
// text their own MarshalText emits. Taking the width from binary.Size ties text
// decoding to the same source as the binary width.
func parseIntText[T int8 | int16 | int32 | int64](text []byte, v *T) error {
	i, err := strconv.ParseInt(string(text), 10, binary.Size(*v)*8)
	if err != nil {
		return err
	}
	*v = T(i)
	return nil
}

// parseUintText is the unsigned twin of parseIntText.
func parseUintText[T uint8 | uint16 | uint32 | uint64](text []byte, v *T) error {
	u, err := strconv.ParseUint(string(text), 10, binary.Size(*v)*8)
	if err != nil {
		return err
	}
	*v = T(u)
	return nil
}
