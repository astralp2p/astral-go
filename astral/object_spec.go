package astral

import "io"

var _ Object = (*ObjectSpec)(nil)

// ObjectSpec describes a field that holds any Object; the value is encoded polymorphically
// (type tag + payload) on the wire.
type ObjectSpec struct{}

func (*ObjectSpec) ObjectType() string { return "astral.blueprint.object_spec" }

func (*ObjectSpec) WriteTo(io.Writer) (int64, error)  { return 0, nil }
func (*ObjectSpec) ReadFrom(io.Reader) (int64, error) { return 0, nil }

// why: plain encoding/json decodes a carrier payload reflectively and ignores keys it does
// not recognise, so a misspelled key inside the {"Type","Object"} envelope decoded to a
// zero-valued carrier with no error and registered a corrupted schema permanently.
// Objectify routes through structValue, which rejects excess fields.
func (s *ObjectSpec) MarshalJSON() ([]byte, error) { return Objectify(s).MarshalJSON() }
func (s *ObjectSpec) UnmarshalJSON(b []byte) error { return Objectify(s).UnmarshalJSON(b) }

// ReferencedType satisfies Spec. ObjectSpec is polymorphic — the concrete type travels with the
// payload as a wire tag, so the schema itself depends on no specific name.
func (*ObjectSpec) ReferencedType() string { return "" }

func init() { _ = Add(&ObjectSpec{}) }
