package astral

import (
	"encoding/json"
	"io"
	"strconv"
)

type Uint8 uint8

func NewUint8(u uint8) *Uint8 {
	return (*Uint8)(&u)
}

// astral:blueprint-ignore
func (Uint8) ObjectType() string {
	return "uint8"
}

func (u Uint8) WriteTo(w io.Writer) (int64, error) {
	return writeFixed(w, uint8(u))
}

func (u *Uint8) ReadFrom(r io.Reader) (int64, error) {
	return readFixed(r, (*uint8)(u))
}

func (u Uint8) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint8(u))
}

func (u *Uint8) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, (*uint8)(u))
}

func (u Uint8) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(u), 10)), nil
}

func (u *Uint8) UnmarshalText(text []byte) error {
	return parseUintText(text, (*uint8)(u))
}

func (u Uint8) String() string {
	return strconv.FormatUint(uint64(u), 10)
}

type Uint16 uint16

func NewUint16(u uint16) *Uint16 {
	return (*Uint16)(&u)
}

// astral:blueprint-ignore
func (Uint16) ObjectType() string {
	return "uint16"
}

func (u Uint16) WriteTo(w io.Writer) (int64, error) {
	return writeFixed(w, uint16(u))
}

func (u *Uint16) ReadFrom(r io.Reader) (int64, error) {
	return readFixed(r, (*uint16)(u))
}

func (u Uint16) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint16(u))
}

func (u *Uint16) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, (*uint16)(u))
}

func (u Uint16) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(u), 10)), nil
}

func (u *Uint16) UnmarshalText(text []byte) error {
	return parseUintText(text, (*uint16)(u))
}

func (u Uint16) String() string {
	return strconv.FormatUint(uint64(u), 10)
}

type Uint32 uint32

func NewUint32(u uint32) *Uint32 {
	return (*Uint32)(&u)
}

// astral:blueprint-ignore
func (Uint32) ObjectType() string {
	return "uint32"
}

func (u Uint32) WriteTo(w io.Writer) (int64, error) {
	return writeFixed(w, uint32(u))
}

func (u *Uint32) ReadFrom(r io.Reader) (int64, error) {
	return readFixed(r, (*uint32)(u))
}

func (u Uint32) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint32(u))
}

func (u *Uint32) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, (*uint32)(u))
}

func (u Uint32) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(u), 10)), nil
}

func (u *Uint32) UnmarshalText(text []byte) error {
	return parseUintText(text, (*uint32)(u))
}

func (u Uint32) String() string {
	return strconv.FormatUint(uint64(u), 10)
}

type Uint64 uint64

func NewUint64(u uint64) *Uint64 {
	return (*Uint64)(&u)
}

// astral:blueprint-ignore
func (Uint64) ObjectType() string {
	return "uint64"
}

func (u Uint64) WriteTo(w io.Writer) (int64, error) {
	return writeFixed(w, uint64(u))
}

func (u *Uint64) ReadFrom(r io.Reader) (int64, error) {
	return readFixed(r, (*uint64)(u))
}

func (u Uint64) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint64(u))
}

func (u *Uint64) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, (*uint64)(u))
}

func (u Uint64) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(u), 10)), nil
}

func (u *Uint64) UnmarshalText(text []byte) error {
	return parseUintText(text, (*uint64)(u))
}

func (u Uint64) String() string {
	return strconv.FormatUint(uint64(u), 10)
}

func init() {
	var (
		u8  Uint8
		u16 Uint16
		u32 Uint32
		u64 Uint64
	)
	MustAdd(&u8, &u16, &u32, &u64)
}
