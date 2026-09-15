package routing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/streams"
)

type OpSpec struct {
	Name       string
	Parameters []query.FieldSpec
}

var _ astral.Object = &OpSpec{}

func (OpSpec) ObjectType() string {
	return "routing.op_spec"
}

// binary
//
// why: OpSpec encodes by hand, and so has no Blueprint. A parameter entry has no registered
// Object Type of its own (routing.op_spec in the spec), and a Blueprint can only describe a
// list of a registered type, so the reflection codec refuses the struct. The layout below is
// the one the reflection codec produced: Name as string32, then a uint32 count, then per
// parameter a 0x01 presence byte, Name and Type as string32, and Required as bool.

func (s OpSpec) WriteTo(w io.Writer) (n int64, err error) {
	n, err = astral.String32(s.Name).WriteTo(w)
	if err != nil {
		return
	}

	m, err := astral.Uint32(len(s.Parameters)).WriteTo(w)
	n += m
	if err != nil {
		return
	}

	for _, p := range s.Parameters {
		m, err = writeParam(w, p)
		n += m
		if err != nil {
			return
		}
	}

	return
}

func (s *OpSpec) ReadFrom(r io.Reader) (n int64, err error) {
	var name astral.String32
	var count astral.Uint32

	n, err = name.ReadFrom(r)
	if err != nil {
		return
	}

	m, err := count.ReadFrom(r)
	n += m
	if err != nil {
		return
	}

	// why: grow per decoded entry rather than reserving count up front, so a forged count
	// costs no allocation beyond the bytes actually present.
	var params []query.FieldSpec
	for range count {
		var p query.FieldSpec
		m, err = readParam(r, &p)
		n += m
		if err != nil {
			return
		}
		params = append(params, p)
	}

	s.Name, s.Parameters = name.String(), params
	return
}

func writeParam(w io.Writer, p query.FieldSpec) (n int64, err error) {
	var presence = astral.Uint8(1)
	return streams.WriteAllTo(w, presence, astral.String32(p.Name), astral.String32(p.Type), astral.Bool(p.Required))
}

func readParam(r io.Reader, p *query.FieldSpec) (n int64, err error) {
	var presence astral.Uint8
	var name, typ astral.String32
	var required astral.Bool

	n, err = presence.ReadFrom(r)
	switch {
	case err != nil:
		return
	case presence != 1:
		return n, fmt.Errorf("invalid presence flag %d", presence)
	}

	m, err := streams.ReadAllFrom(r, &name, &typ, &required)
	n += m
	if err != nil {
		return
	}

	*p = query.FieldSpec{Name: name.String(), Type: typ.String(), Required: bool(required)}
	return
}

// json
//
// why: the reflection codec emitted keys in alphabetical order and an empty array for no
// parameters, and rejected unknown keys. The mirror types keep all three.

type opSpecJSON struct {
	Name       string
	Parameters []paramJSON
}

type paramJSON struct {
	Name     string
	Required bool
	Type     string
}

func (s OpSpec) MarshalJSON() ([]byte, error) {
	var j = opSpecJSON{Name: s.Name, Parameters: make([]paramJSON, 0, len(s.Parameters))}
	for _, p := range s.Parameters {
		j.Parameters = append(j.Parameters, paramJSON{Name: p.Name, Required: p.Required, Type: p.Type})
	}
	return json.Marshal(j)
}

func (s *OpSpec) UnmarshalJSON(data []byte) error {
	var j opSpecJSON

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&j); err != nil {
		return err
	}

	var params []query.FieldSpec
	for _, p := range j.Parameters {
		params = append(params, query.FieldSpec{Name: p.Name, Type: p.Type, Required: p.Required})
	}

	s.Name, s.Parameters = j.Name, params
	return nil
}

// ...

func init() {
	astral.MustAdd(&OpSpec{})
}
