package astral

import "io"

var _ Object = (*SliceSpec)(nil)

// SliceSpec describes a field whose value is a Slice of objects. An empty Type means heterogeneous
// elements (each carries its own type tag).
type SliceSpec struct {
	Type String16
}

func (*SliceSpec) ObjectType() string { return "astral.blueprint.slice_spec" }

func (s *SliceSpec) WriteTo(w io.Writer) (int64, error)  { return Objectify(s).WriteTo(w) }
func (s *SliceSpec) ReadFrom(r io.Reader) (int64, error) { return Objectify(s).ReadFrom(r) }

// why: plain encoding/json decodes a carrier payload reflectively and ignores keys it does
// not recognise, so a misspelled key inside the {"Type","Object"} envelope decoded to a
// zero-valued carrier with no error and registered a corrupted schema permanently.
// Objectify routes through structValue, which rejects excess fields.
func (s *SliceSpec) MarshalJSON() ([]byte, error) { return Objectify(s).MarshalJSON() }
func (s *SliceSpec) UnmarshalJSON(b []byte) error { return Objectify(s).UnmarshalJSON(b) }

// ReferencedType satisfies Spec. SliceSpec depends on its element Type for closure validation;
// empty Type (heterogeneous) declares no dependency.
func (s *SliceSpec) ReferencedType() string { return s.Type.String() }

func init() { _ = Add(&SliceSpec{}) }
