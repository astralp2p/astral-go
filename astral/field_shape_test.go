package astral

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"
)

// The reflection codec refuses every struct field BlueprintFromType cannot describe, on all
// four paths. Each shape below encoded without error before, and derived no Blueprint.

type shapeOf[T any] struct{ V T }

type plainInner struct{ A Uint8 }

// PlainInnerForShapeTest is exported so it embeds as an exported field, the shape both sides
// read; an unexported embedded field is skipped by both.
type PlainInnerForShapeTest struct{ A Uint8 }

type embedsPlain struct {
	PlainInnerForShapeTest
	B Uint8
}

type ObjectInnerForShapeTest struct{ A Uint8 }

func (ObjectInnerForShapeTest) ObjectType() string                   { return "test.shape_inner" }
func (o ObjectInnerForShapeTest) WriteTo(w io.Writer) (int64, error) { return Objectify(&o).WriteTo(w) }
func (o *ObjectInnerForShapeTest) ReadFrom(r io.Reader) (int64, error) {
	return Objectify(o).ReadFrom(r)
}

type embedsObject struct {
	ObjectInnerForShapeTest
	B Uint8
}

func refusedShapes() map[string]any {
	return map[string]any{
		"bool":              &shapeOf[bool]{},
		"uint8":             &shapeOf[uint8]{},
		"uint32":            &shapeOf[uint32]{},
		"int64":             &shapeOf[int64]{},
		"float64":           &shapeOf[float64]{},
		"string":            &shapeOf[string]{},
		"[]byte":            &shapeOf[[]byte]{},
		"[2]byte":           &shapeOf[[2]byte]{},
		"map value string":  &shapeOf[map[string]string]{},
		"pointer to plain":  &shapeOf[*plainInner]{},
		"named plain":       &shapeOf[plainInner]{},
		"slice of plain":    &shapeOf[[]plainInner]{},
		"pointer to Object": &shapeOf[*Object]{},
		"embedded plain":    &embedsPlain{},
	}
}

func TestFieldShape_RefusesUndescribableFields(t *testing.T) {
	for name, v := range refusedShapes() {
		t.Run(name, func(t *testing.T) {
			o := Objectify(v)

			_, err := o.WriteTo(io.Discard)
			assertUndescribable(t, "WriteTo", err)

			_, err = o.ReadFrom(bytes.NewReader(make([]byte, 16)))
			assertUndescribable(t, "ReadFrom", err)

			_, err = o.MarshalJSON()
			assertUndescribable(t, "MarshalJSON", err)

			assertUndescribable(t, "UnmarshalJSON", o.UnmarshalJSON([]byte(`{}`)))
		})
	}
}

func assertUndescribable(t *testing.T, path string, err error) {
	t.Helper()
	if !errors.Is(err, ErrUndescribableField) {
		t.Errorf("%s: want ErrUndescribableField, got %v", path, err)
	}
}

// Every refused shape has an astral replacement the codec accepts, and the replacement writes
// the bytes the plain shape wrote before the codec refused it. The hex is what the plain shape
// produced at astral-go 6ea26c7.
func TestFieldShape_ReplacementsKeepTheBytes(t *testing.T) {
	cases := []struct {
		name  string
		value any
		hex   string
	}{
		{"bool -> Bool", &shapeOf[Bool]{V: true}, "01"},
		{"uint8 -> Uint8", &shapeOf[Uint8]{V: 7}, "07"},
		{"uint32 -> Uint32", &shapeOf[Uint32]{V: 7}, "00000007"},
		{"string -> String32", &shapeOf[String32]{V: "ab"}, "000000026162"},
		{"[]byte -> []Uint8", &shapeOf[[]Uint8]{V: []Uint8{0xaa, 0xbb}}, "0000000201aa01bb"},
		{"embedded plain -> embedded Object", &embedsObject{ObjectInnerForShapeTest{1}, 2}, "0102"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := Objectify(c.value).WriteTo(&buf); err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(buf.Bytes()); got != c.hex {
				t.Fatalf("want %s, got %s", c.hex, got)
			}
		})
	}
}

// Bytes written for a plain []byte field decode into []Uint8.
func TestFieldShape_PlainByteSliceBytesDecodeIntoUint8Slice(t *testing.T) {
	raw, _ := hex.DecodeString("0000000201aa01bb")

	var dst shapeOf[[]Uint8]
	if _, err := Objectify(&dst).ReadFrom(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	if len(dst.V) != 2 || dst.V[0] != 0xaa || dst.V[1] != 0xbb {
		t.Fatalf("want [aa bb], got %x", dst.V)
	}
}
