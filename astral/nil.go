package astral

// Nil is a pseudo-object that represents the nil value.
type Nil struct {
	EmptyObject
}

var _ Object = &Nil{}

// nilTypeName is the registered type of the Nil marker. It stays a first-class
// top-level object — objects.new answers with it — but inside a polymorphic slot
// absence is spelled as the zero-length type tag, and decoders accept this name
// there only as a legacy alias.
const nilTypeName = "nil"

func (Nil) ObjectType() string { return nilTypeName }

func init() {
	MustAdd(&Nil{})
}
