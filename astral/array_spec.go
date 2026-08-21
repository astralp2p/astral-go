package astral

import "io"

var _ Object = (*ArraySpec)(nil)

// ArraySpec describes a field whose value is a fixed-length array of objects. Length is part of
// the schema, so the wire format omits a count prefix. An empty Type means heterogeneous elements
// (each carries its own type tag).
type ArraySpec struct {
	Type   String16
	Length Uint32
}

func (*ArraySpec) ObjectType() string { return "astral.blueprint.array_spec" }

func (s *ArraySpec) WriteTo(w io.Writer) (int64, error)  { return Objectify(s).WriteTo(w) }
func (s *ArraySpec) ReadFrom(r io.Reader) (int64, error) { return Objectify(s).ReadFrom(r) }

// why: plain encoding/json decodes a carrier payload reflectively and ignores keys it does
// not recognise, so a misspelled key inside the {"Type","Object"} envelope decoded to a
// zero-valued carrier with no error and registered a corrupted schema permanently.
// Objectify routes through structValue, which rejects excess fields.
func (s *ArraySpec) MarshalJSON() ([]byte, error) { return Objectify(s).MarshalJSON() }
func (s *ArraySpec) UnmarshalJSON(b []byte) error { return Objectify(s).UnmarshalJSON(b) }

// ReferencedType satisfies Spec. ArraySpec depends on its element Type for closure validation;
// empty Type (heterogeneous) declares no dependency.
func (s *ArraySpec) ReferencedType() string { return s.Type.String() }

func init() { MustAdd(&ArraySpec{}) }
