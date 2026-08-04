package astral

import "io"

var _ Object = (*PtrSpec)(nil)

// PtrSpec describes an optional field. Wire layout: [Bool nil-flag][inner payload if non-nil].
type PtrSpec struct {
	Type String16
}

func (*PtrSpec) ObjectType() string { return "astral.blueprint.ptr_spec" }

func (s *PtrSpec) WriteTo(w io.Writer) (int64, error)  { return Objectify(s).WriteTo(w) }
func (s *PtrSpec) ReadFrom(r io.Reader) (int64, error) { return Objectify(s).ReadFrom(r) }

// why: plain encoding/json decodes a carrier payload reflectively and ignores keys it does
// not recognise, so a misspelled key inside the {"Type","Object"} envelope decoded to a
// zero-valued carrier with no error and registered a corrupted schema permanently.
// Objectify routes through structValue, which rejects excess fields.
func (s *PtrSpec) MarshalJSON() ([]byte, error) { return Objectify(s).MarshalJSON() }
func (s *PtrSpec) UnmarshalJSON(b []byte) error { return Objectify(s).UnmarshalJSON(b) }

// ReferencedType satisfies Spec. PtrSpec wraps an optional value of Type; that name must be
// registered before this Blueprint for closure validation to pass.
func (s *PtrSpec) ReferencedType() string { return s.Type.String() }

func init() { MustAdd(&PtrSpec{}) }
