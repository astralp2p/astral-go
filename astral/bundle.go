package astral

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sync"
)

var _ Object = &Bundle{}

// Bundle is an ordered collection of unique objects of various types.
type Bundle struct {
	objects []Object
	index   map[string]int
	mu      sync.Mutex
}

func (*Bundle) ObjectType() string { return "bundle" }

// NewBundle returns a new Bundle instance. objects can be nil.
func NewBundle() *Bundle {
	return &Bundle{}
}

// Append appends objects to the bundle.
func (b *Bundle) Append(objects ...Object) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.index == nil {
		b.index = make(map[string]int)
	}

	for _, o := range objects {
		err := b.append(o)
		if err != nil {
			return err
		}
	}

	return nil
}

// Fetch fetches the object from the Bundle. Returns nil if the object is not in the Bundle.
func (b *Bundle) Fetch(objectID ObjectID) Object {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.index == nil {
		return nil
	}

	idstr := objectID.String()
	idx, ok := b.index[idstr]
	if !ok {
		return nil
	}

	return b.objects[idx]
}

// WriteTo writes Bundle's payload to the writer.
//
// why: a pointer receiver. The value receiver copied the mutex, so it locked a copy and
// provided no exclusion at all — go vet reports it.
func (b *Bundle) WriteTo(w io.Writer) (n int64, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ow, gerr := enterWriter(w, frameName("bundle"))
	defer ow.exit()
	if gerr != nil {
		return 0, gerr
	}
	w = ow

	// write object count
	n, err = Uint32(len(b.objects)).WriteTo(w)
	if err != nil {
		return
	}

	// write objects
	var m int64
	for _, o := range b.objects {
		// why: the member block is length-prefixed, so its payload must exist before its
		// length can be written — the staging buffer is required by the wire format. The
		// derived writer shares this call's ledger so buffering cannot reset the guard.
		var raw = &bytes.Buffer{}
		var buf = ow.subWriter(raw)

		// write the type to the buffer
		_, err = String8(o.ObjectType()).WriteTo(buf)
		if err != nil {
			return
		}

		// write the payload to the buffer
		_, err = o.WriteTo(buf)
		if err != nil {
			return
		}

		// write the length-encoded buffer to the writer
		m, err = Bytes32(raw.Bytes()).WriteTo(w)
		n += m
		if err != nil {
			return
		}
	}

	return
}

// ReadFrom reads Bundle's payload from the reader.
func (b *Bundle) ReadFrom(r io.Reader) (n int64, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	or, gerr := enterReader(r, frameName("bundle"))
	defer or.exit()
	if gerr != nil {
		return 0, gerr
	}
	r = or

	// read object count
	var count Uint32
	n, err = count.ReadFrom(r)
	if err != nil {
		return
	}

	// why: decode into locals and commit at the end. Resetting the receiver up front
	// destroyed its previous contents on a read that then failed, and left a partially
	// filled bundle that Objects, String and Fetch all served as if it were whole.
	var objects []Object
	index := make(map[string]int)

	var m int64
	var o Object
	for i := 0; i < int(count); i++ {
		// read a 32-bit length-encoded buffer
		var buf = Bytes32{}
		m, err = buf.ReadFrom(r)
		n += m
		if err != nil {
			return
		}

		// why: or.sub shares this call's ledger. A bare bytes.Reader started a fresh
		// one, so a Bundle anywhere in a nested payload reset the depth guard for
		// everything below it — and it dropped the per-call Blueprints registry too.
		o, _, err = Decode(or.sub(buf), WithBlueprints(or.resolve()))
		if err != nil {
			return
		}

		objects, index, err = appendUnique(objects, index, o)
		if err != nil {
			return
		}
	}

	b.objects, b.index = objects, index

	return
}

// appendUnique adds o to a bundle under construction, rejecting a duplicate Object ID.
func appendUnique(objects []Object, index map[string]int, o Object) ([]Object, map[string]int, error) {
	objectID, err := ResolveObjectID(o)
	if err != nil {
		return objects, index, err
	}

	idstr := objectID.String()
	if _, found := index[idstr]; found {
		return objects, index, fmt.Errorf("duplicate object")
	}

	index[idstr] = len(objects)
	return append(objects, o), index, nil
}

// Objects returns a copy of the underlying list of objects
func (b *Bundle) Objects() []Object {
	b.mu.Lock()
	defer b.mu.Unlock()

	return slices.Clone(b.objects)
}

func (b *Bundle) String() string {
	return fmt.Sprintf("bundle (%d objects)", len(b.objects))
}

func (b *Bundle) MarshalJSON() ([]byte, error) {
	var err error
	var list []JSONAdapter
	for _, o := range b.objects {
		j := JSONAdapter{
			Type: o.ObjectType(),
		}
		j.Object, err = json.Marshal(o)
		if err != nil {
			return nil, err
		}

		list = append(list, j)
	}

	return json.Marshal(list)
}

func (a *Bundle) UnmarshalJSON(bytes []byte) error {
	var jlist []JSONAdapter
	a.index = make(map[string]int)

	if err := json.Unmarshal(bytes, &jlist); err != nil {
		return err
	}

	for _, j := range jlist {
		obj := New(j.Type)
		if obj == nil {
			return fmt.Errorf("%w: %s", ErrBlueprintNotFound, j.Type)
		}

		var err error
		if j.Object != nil {
			err = json.Unmarshal(j.Object, &obj)
			if err != nil {
				return err
			}
		}

		// append object will also handle index
		err = a.append(obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bundle) append(object Object) error {
	objectID, err := ResolveObjectID(object)
	if err != nil {
		return fmt.Errorf("error resolving object id: %w", err)
	}

	// skip duplicates
	idstr := objectID.String()
	if _, found := b.index[idstr]; found {
		return fmt.Errorf("duplicate object")
	}

	b.objects = append(b.objects, object)
	b.index[idstr] = len(b.objects) - 1
	return nil
}

// SelectByType selects objects of the parameter type from a generic list of Objects
func SelectByType[T Object](objects []Object) (list []T) {
	for _, o := range objects {
		if o, ok := o.(T); ok {
			list = append(list, o)
		}
	}
	return
}

// First returns the first object of the parameter type from a generic list of Objects
func First[T Object](objects []Object) (object T, found bool) {
	for _, o := range objects {
		if o, ok := o.(T); ok {
			return o, true
		}
	}
	return
}

// Fetch fetches a type-cast object from the bundle
func Fetch[T Object](bundle *Bundle, objectID ObjectID) (object T, found bool) {
	object, found = bundle.Fetch(objectID).(T)
	return
}

func init() {
	_ = Add(&Bundle{})
}
