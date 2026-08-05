// Package defects reproduces every shape the rule must catch. Each of the four is a
// defect this module shipped; the comment on each names it. testdata is invisible to
// the go tool, so nothing here is built or vetted.
package defects

import (
	"io"
	"reflect"
)

type identity struct{ key []byte }

func (identity) WriteTo(w io.Writer) (int64, error)   { return 0, nil }
func (*identity) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

func writeAll(w io.Writer, sources ...any) (int64, error) { return 0, nil }
func readAll(r io.Reader, targets ...any) (int64, error)  { return 0, nil }
func objectify(v any) io.ReadWriter                       { return nil }

// nodeInfo is the shipped NodeInfo crash. A zero value carries a nil *identity, which
// still satisfies io.WriterTo, so the helper calls a value-receiver WriteTo on nil.
type nodeInfo struct {
	Alias    string
	Identity *identity
}

func (nodeInfo) ObjectType() string { return "mod.nodes.node_info" }

func (info nodeInfo) WriteTo(w io.Writer) (n int64, err error) {
	return writeAll(w, info.Alias, info.Identity)
}

func (info *nodeInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return readAll(r, &info.Alias, info.Identity)
}

// stringView is the byte-identical twin the original audit missed. It is never
// registered, so no runtime sweep materializes it.
type stringView struct {
	Style renderer
	str   *text
}

type renderer struct{}
type text struct{}

func (*text) WriteTo(w io.Writer) (int64, error)  { return 0, nil }
func (*text) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

func (stringView) ObjectType() string { return "string32" }

func (v stringView) WriteTo(w io.Writer) (n int64, err error) {
	return v.str.WriteTo(w)
}

func (v *stringView) ReadFrom(r io.Reader) (n int64, err error) {
	return v.str.ReadFrom(r)
}

// endpointMapping is kcp.EndpointLocalMapping, whose ReadFrom decoded into a copy the
// return discarded, and whose Objectify call panicked before it got that far.
type endpointMapping struct {
	Address string
	Port    uint16
}

func (e endpointMapping) ObjectType() string { return "mod.kcp.endpoint_local_mapping" }

// WriteTo takes &e on a value receiver too, and is correct: encoding reads the copy and
// needs nothing back from it. It stays here to pin that the rule reads ReadFrom only.
func (e endpointMapping) WriteTo(w io.Writer) (n int64, err error) {
	return objectify(&e).(io.WriterTo).WriteTo(w)
}

func (e endpointMapping) ReadFrom(r io.Reader) (n int64, err error) {
	return objectify(&e).(io.ReaderFrom).ReadFrom(r)
}

// runtimeSlice is one of the three Runtime* carriers. Its zero value reaches the codec
// through reflect.New on every decode, and the zero reflect.Value has no type, so every
// accessor on it panics.
type runtimeSlice struct {
	ptr      reflect.Value
	elemName string
}

func (*runtimeSlice) ObjectType() string { return "slice" }

func (s *runtimeSlice) WriteTo(w io.Writer) (int64, error) {
	return sliceCodec{Value: s.ptr.Elem()}.WriteTo(w)
}

func (s *runtimeSlice) ReadFrom(r io.Reader) (int64, error) {
	return sliceCodec{Value: s.ptr.Elem()}.ReadFrom(r)
}

type sliceCodec struct{ reflect.Value }

func (sliceCodec) WriteTo(w io.Writer) (int64, error)  { return 0, nil }
func (sliceCodec) ReadFrom(r io.Reader) (int64, error) { return 0, nil }

// lateGuard proves the field after it has already loaded through it, which is the same
// crash with a consolation prize. The presence of a nil check is not the question;
// whether it runs first is.
type lateGuard struct {
	Identity *identity
}

func (lateGuard) ObjectType() string { return "mod.test.late_guard" }

func (g *lateGuard) WriteTo(w io.Writer) (n int64, err error) {
	n, err = writeAll(w, g.Identity)
	if g.Identity == nil {
		return 0, nil
	}
	return
}
