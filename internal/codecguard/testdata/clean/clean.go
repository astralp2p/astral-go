// Package clean holds the shapes the rule must stay silent on. Most are measured: the
// reflective carriers below are 20 of the 23 hits the naive value-receiver rule
// produced, and the two guarded codecs are what the first draft of the pointer rule
// reported before it learned to follow a copy and to read an assignment as an
// initialisation. testdata is invisible to the go tool, so nothing here is built.
package clean

import (
	"io"
	"reflect"
)

type identity struct{ key []byte }

func (identity) WriteTo(w io.Writer) (int64, error)   { return 0, nil }
func (*identity) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

func writeAll(w io.Writer, sources ...any) (int64, error) { return 0, nil }
func readAll(r io.Reader, targets ...any) (int64, error)  { return 0, nil }

// boolCarrier wraps a reflect.Value, for which a value receiver is correct: the Value
// is a handle and SetBool writes through it. The naive "a value-receiver ReadFrom is an
// error" rule reported this and its siblings, and was wrong every time.
type boolCarrier struct {
	reflect.Value
}

func (b boolCarrier) ObjectType() string { return "bool" }

func (b boolCarrier) WriteTo(w io.Writer) (n int64, err error) {
	_ = b.Bool()
	return 1, nil
}

func (b boolCarrier) ReadFrom(r io.Reader) (n int64, err error) {
	b.SetBool(true)
	return 1, nil
}

// emptyMsg carries no payload, so a value-receiver ReadFrom has nothing to lose.
type emptyMsg struct{}

func (emptyMsg) ObjectType() string                    { return "mod.objects.commit_msg" }
func (c emptyMsg) WriteTo(w io.Writer) (int64, error)  { return 0, nil }
func (c emptyMsg) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

// guardedNodeInfo is the fix that shipped: WriteTo proves a copy of the field, ReadFrom
// assigns the field before reading into it.
type guardedNodeInfo struct {
	Alias    string
	Identity *identity
}

func (guardedNodeInfo) ObjectType() string { return "mod.nodes.node_info" }

func (info guardedNodeInfo) WriteTo(w io.Writer) (n int64, err error) {
	id := info.Identity
	if id == nil {
		id = &identity{}
	}
	return writeAll(w, info.Alias, id)
}

func (info *guardedNodeInfo) ReadFrom(r io.Reader) (n int64, err error) {
	info.Identity = &identity{}
	return readAll(r, &info.Alias, info.Identity)
}

// guardedStringView proves the field directly, in both directions.
type guardedStringView struct {
	str *text
}

type text struct{}

func newText() *text                              { return &text{} }
func (*text) WriteTo(w io.Writer) (int64, error)  { return 0, nil }
func (*text) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

func (guardedStringView) ObjectType() string { return "string32" }

func (v guardedStringView) WriteTo(w io.Writer) (n int64, err error) {
	if v.str == nil {
		return 0, nil
	}
	return v.str.WriteTo(w)
}

func (v *guardedStringView) ReadFrom(r io.Reader) (n int64, err error) {
	if v.str == nil {
		v.str = newText()
	}
	return v.str.ReadFrom(r)
}

// guardedRuntimeSlice asks the only question a reflect.Value answers before its type is
// known.
type guardedRuntimeSlice struct {
	ptr reflect.Value
}

func (*guardedRuntimeSlice) ObjectType() string { return "slice" }

func (s *guardedRuntimeSlice) WriteTo(w io.Writer) (int64, error) {
	if !s.ptr.IsValid() {
		return 0, nil
	}
	return sliceCodec{Value: s.ptr.Elem()}.WriteTo(w)
}

func (s *guardedRuntimeSlice) ReadFrom(r io.Reader) (int64, error) {
	if !s.ptr.IsValid() {
		return 0, nil
	}
	return sliceCodec{Value: s.ptr.Elem()}.ReadFrom(r)
}

type sliceCodec struct{ reflect.Value }

func (sliceCodec) WriteTo(w io.Writer) (int64, error)  { return 0, nil }
func (sliceCodec) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

// addressed only takes the field's address, which reads the slot and not the value
// behind it. A nil pointer survives that.
type addressed struct {
	Next *identity
}

func (addressed) ObjectType() string { return "mod.test.addressed" }

func (a *addressed) WriteTo(w io.Writer) (n int64, err error) {
	return writeAll(w, &a.Next)
}

func (a *addressed) ReadFrom(r io.Reader) (n int64, err error) {
	return readAll(r, &a.Next)
}

// shadowed names its copy after the field. Without skipping the identifier that names a
// member, the walk reads the field a second time through the selector's own Sel.
type shadowed struct {
	str *text
}

func (shadowed) ObjectType() string { return "mod.test.shadowed" }

func (v *shadowed) WriteTo(w io.Writer) (n int64, err error) {
	str := v.str
	if str == nil {
		return 0, nil
	}
	return str.WriteTo(w)
}

func (v *shadowed) ReadFrom(r io.Reader) (n int64, err error) {
	if v.str == nil {
		v.str = newText()
	}
	return v.str.ReadFrom(r)
}

// notAnObject declares no ObjectType, so its WriteTo is not a codec and the rule does
// not read it. Narrowing to registered wire types is what keeps the rule quiet.
type notAnObject struct {
	Inner *identity
}

func (p *notAnObject) WriteTo(w io.Writer) (n int64, err error) {
	return p.Inner.WriteTo(w)
}

func (p *notAnObject) ReadFrom(r io.Reader) (n int64, err error) {
	return p.Inner.ReadFrom(r)
}
