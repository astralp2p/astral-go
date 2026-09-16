package routing

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// OpParam is one parameter an operation accepts: routing.op_param.
//
// why: a Blueprint describes a list by naming its element type, so the parameter entry
// needs a registered type of its own. The wire form is the one the plain struct produced.
type OpParam struct {
	Name     astral.String32
	Type     astral.String32
	Required astral.Bool
}

func (OpParam) ObjectType() string { return "routing.op_param" }

func (p OpParam) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&p).WriteTo(w)
}

func (p *OpParam) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(p).ReadFrom(r)
}

type OpSpec struct {
	Name       astral.String32
	Parameters []OpParam
}

var _ astral.Object = &OpSpec{}

func (OpSpec) ObjectType() string {
	return "routing.op_spec"
}

// binary

func (s OpSpec) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&s).WriteTo(w)
}

func (s *OpSpec) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(s).ReadFrom(r)
}

// json

func (s OpSpec) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&s).MarshalJSON()
}

func (s *OpSpec) UnmarshalJSON(bytes []byte) error {
	return astral.Objectify(s).UnmarshalJSON(bytes)
}

// ...

func init() {
	astral.MustAdd(&OpParam{})
	astral.MustAdd(&OpSpec{})
}
