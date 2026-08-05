package astral

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type interfaceValue struct {
	reflect.Value
}

var _ Object = &interfaceValue{}

// astral:blueprint-ignore
func (i interfaceValue) ObjectType() string {
	return ""
}

// IsAbsent reports whether the slot carries no value. Three forms mean the same
// thing — interface-nil, a typed nil pointer, and the *Nil marker the runtime
// Blueprint codec installs as a polymorphic field's zero — and the spec spells all
// of them as the zero-length type tag (topics/binary-encoding.md, topics/codec.md).
func (i interfaceValue) IsAbsent() bool {
	if !i.IsValid() || i.IsNil() || i.IsElemNilPtr() {
		return true
	}
	_, isNil := i.Interface().(*Nil)
	return isNil
}

func (i interfaceValue) WriteTo(w io.Writer) (n int64, err error) {
	ow, gerr := enterWriter(w, frameName("interface"))
	defer ow.exit()
	if gerr != nil {
		return 0, gerr
	}
	w = ow

	if i.IsAbsent() {
		err = binary.Write(w, ByteOrder, uint8(0)) // zero-length type means nil
		if err == nil {
			return 1, nil
		}
		return
	}

	var objectType string
	var objectWriter io.WriterTo

	switch i.Elem().Kind() {
	case reflect.Ptr:
		v := ptrValue{Value: i.Elem(), skipNilFlag: true}
		objectWriter = v
		objectType = v.ObjectType()

	case reflect.String, reflect.Slice:
		// this is a special case needed to handle various String* and Bytes* alias types
		ow, wok := i.Elem().Interface().(io.WriterTo)
		ot, tok := i.Elem().Interface().(ObjectTyper)
		if wok && tok {
			objectWriter = ow
			objectType = ot.ObjectType()
			break
		}

		fallthrough
	default:
		o, err := objectify(i.Elem())
		if err != nil {
			return n, err
		}

		objectType = o.ObjectType()
		objectWriter = o
	}

	if objectType == "" {
		return n, errors.New("interface contains an untyped object")
	}

	var m int64
	m, err = String8(objectType).WriteTo(w)
	n += m
	if err != nil {
		return
	}

	m, err = objectWriter.WriteTo(w)
	n += m

	return
}

func (i interfaceValue) ReadFrom(r io.Reader) (n int64, err error) {
	or, gerr := enterReader(r, frameName("interface"))
	defer or.exit()
	if gerr != nil {
		return 0, gerr
	}
	r = or

	var objectType string
	m, err := (*String8)(&objectType).ReadFrom(r)
	n += m
	if err != nil {
		return
	}

	// why: the zero-length tag is the spec's spelling of absence. nilTypeName is
	// accepted alongside it for peers written before that was settled.
	if len(objectType) == 0 || objectType == nilTypeName {
		i.Set(reflect.Zero(i.Type()))
		return
	}

	// why: a polymorphic field is the one slot whose type is not known until the bytes
	// arrive, so it is precisely where a per-call registry has to be consulted. The
	// package-level New reads defaultBlueprints, which meant a registry passed with
	// WithBlueprints reached a struct's own fields and was then dropped at the first
	// polymorphic one. resolve() falls back to defaultBlueprints when no registry was
	// threaded, and a child registry walks its parent chain, so nothing that resolved
	// before stops resolving.
	o := or.resolve().New(objectType)
	if o == nil {
		return n, fmt.Errorf("%w: %w: %s", ErrStreamCorrupted, ErrBlueprintNotFound, objectType)
	}

	if !reflect.ValueOf(o).CanConvert(i.Type()) {
		err = fmt.Errorf("cannot convert %s to %s", reflect.TypeOf(o), i.Type())
		return
	}

	m, err = o.ReadFrom(r)
	n += m
	if err != nil {
		return
	}

	i.Set(reflect.ValueOf(o).Convert(i.Type()))

	return
}

func (i interfaceValue) MarshalJSON() ([]byte, error) {
	if i.IsAbsent() {
		return jsonNull, nil
	}

	if _, ok := i.Interface().(*UnparsedObject); ok {
		return nil, errors.New("interface contains an unparsed object")
	}

	o, err := objectify(i.Elem())
	if err != nil {
		return nil, err
	}

	if o.ObjectType() == "" {
		return nil, errors.New("object behind interface has no type")
	}

	jsonBytes, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return json.Marshal(JSONAdapter{
		Type:   o.ObjectType(),
		Object: jsonBytes,
	})
}

func (i interfaceValue) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, jsonNull) {
		i.SetZero()
		return nil
	}

	var j JSONAdapter
	err := json.Unmarshal(data, &j)
	if err != nil {
		return err
	}

	// The JSON twin of the resolution ReadFrom does, and it cannot be fixed the same
	// way: UnmarshalJSON's signature carries no configuration, so there is nowhere to
	// thread a per-call registry. Every JSON type-name resolution in this package has
	// the same ceiling — here, unmarshalFieldJSON, unmarshalRuntimeBlueprintPtr and
	// Bundle — so WithBlueprints is a binary-path facility today. Widening it needs an
	// API decision, not a call-site change.
	o := New(j.Type)
	if o == nil {
		return fmt.Errorf("%w: %s", ErrBlueprintNotFound, j.Type)
	}

	// why: the envelope names any registered type, so a slot narrower than astral.Object
	// (e.g. Field.Spec) must reject a non-implementing type here — the guard ReadFrom has
	// at its Convert. Without it a hostile envelope panics reflect.Value.Convert, and JSON
	// channels make the slot network-reachable (objects.register_blueprint -in json).
	if !reflect.ValueOf(o).CanConvert(i.Type()) {
		return fmt.Errorf("cannot convert %s to %s", reflect.TypeOf(o), i.Type())
	}

	if j.Object != nil {
		err = json.Unmarshal(j.Object, o)
		if err != nil {
			return err
		}
	}

	i.Set(reflect.ValueOf(o).Convert(i.Type()))

	return nil
}

func (i interfaceValue) IsElemNilPtr() bool {
	return i.Elem().Kind() == reflect.Ptr && i.Elem().IsNil()
}
