package astral

import "io"

var _ Object = (*RefSpec)(nil)

// RefSpec describes a field whose value is another registered Object type, encoded inline (no type tag).
type RefSpec struct {
	Type String16
}

func (*RefSpec) ObjectType() string { return "astral.blueprint.ref_spec" }

func (s *RefSpec) WriteTo(w io.Writer) (int64, error)  { return Objectify(s).WriteTo(w) }
func (s *RefSpec) ReadFrom(r io.Reader) (int64, error) { return Objectify(s).ReadFrom(r) }

// why: plain encoding/json decodes a carrier payload reflectively and ignores keys it does
// not recognise, so a misspelled key inside the {"Type","Object"} envelope decoded to a
// zero-valued carrier with no error and registered a corrupted schema permanently.
// Objectify routes through structValue, which rejects excess fields.
func (s *RefSpec) MarshalJSON() ([]byte, error) { return Objectify(s).MarshalJSON() }
func (s *RefSpec) UnmarshalJSON(b []byte) error { return Objectify(s).UnmarshalJSON(b) }

// ReferencedType satisfies Spec. RefSpec inlines another registered Object identified by Type;
// that name must be registered before this Blueprint for closure validation to pass.
func (s *RefSpec) ReferencedType() string { return s.Type.String() }

func init() { _ = Add(&RefSpec{}) }
