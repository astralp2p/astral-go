package astral

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

type sliceValue struct {
	reflect.Value
}

var _ Object = &sliceValue{}

// astral:blueprint-ignore
func (a sliceValue) ObjectType() string {
	return ""
}

func (a sliceValue) WriteTo(w io.Writer) (n int64, err error) {
	ow, gerr := enterWriter(w, frameName("slice"))
	defer ow.exit()
	if gerr != nil {
		return 0, gerr
	}
	w = ow

	var o Object
	var m int64

	err = binary.Write(w, ByteOrder, uint32(a.Len()))
	if err != nil {
		return
	}
	n += 4

	flagged := elemNeedsPresenceFlag(a.Type().Elem())
	for i := range a.Len() {
		o, err = objectify(a.Index(i))
		if err != nil {
			return
		}
		if flagged {
			if _, err = w.Write(presenceFlagOne); err != nil {
				return
			}
			n++
		}
		m, err = o.WriteTo(w)
		n += m
		if err != nil {
			return
		}
	}
	return
}

func (a sliceValue) ReadFrom(r io.Reader) (n int64, err error) {
	or, gerr := enterReader(r, frameName("slice"))
	defer or.exit()
	if gerr != nil {
		return 0, gerr
	}
	r = or

	var o Object
	var m int64
	var l uint32

	err = binary.Read(r, ByteOrder, &l)
	if err != nil {
		return
	}
	// why: the four length bytes were consumed but never counted, so every slice
	// under-reported by four and a type embedding one reported a smaller ReadFrom
	// than WriteTo for the same object. WriteTo counts them.
	n += 4

	// why: l is the peer's claim, not a measurement. Reserving l elements up front
	// let a four-byte count name an arbitrarily large allocation, so the slice is
	// grown as elements arrive and the destination is set only once the whole run
	// reads clean.
	slice := reflect.MakeSlice(a.Type(), 0, elemCap(l))
	elemType := a.Type().Elem()
	flagged := elemNeedsPresenceFlag(elemType)

	for range int(l) {
		elem := reflect.New(elemType).Elem()

		o, err = objectify(elem)
		if err != nil {
			return
		}
		if flagged {
			m, err = consumePresenceFlag(r)
			n += m
			if err != nil {
				return
			}
		}
		m, err = o.ReadFrom(r)
		n += m
		if err != nil {
			return
		}

		slice = reflect.Append(slice, elem)
	}

	a.Set(slice)
	return
}

func (a sliceValue) MarshalJSON() (_ []byte, err error) {
	// why: non-nil allocation so an empty slice marshals to `[]` rather than `null`,
	// matching external-JSON consumer expectations (schema validators, jq, jsonb queries).
	arr := make([]json.RawMessage, 0, a.Len())
	var o Object
	var raw []byte

	for i := range a.Len() {
		o, err = objectify(a.Index(i))
		if err != nil {
			return
		}

		m, ok := o.(json.Marshaler)
		if !ok {
			return nil, errors.New("object does not implement json encoding")
		}

		raw, err = m.MarshalJSON()
		if err != nil {
			return
		}

		arr = append(arr, raw)
	}

	return json.Marshal(arr)
}

func (a sliceValue) UnmarshalJSON(bytes []byte) error {
	var arr []json.RawMessage

	err := json.Unmarshal(bytes, &arr)
	if err != nil {
		return err
	}

	a.Set(reflect.MakeSlice(a.Type(), len(arr), len(arr)))

	for i, raw := range arr {
		o, err := objectify(a.Index(i))
		if err != nil {
			return err
		}

		m, ok := o.(json.Unmarshaler)
		if !ok {
			return errors.New("object does not implement json encoding")
		}

		err = m.UnmarshalJSON(raw)
		if err != nil {
			return err
		}
	}

	return nil
}
