package styles

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

type StringView struct {
	Style Renderer
	str   *astral.String32
}

var _ astral.Object = &StringView{}

// astral:blueprint-ignore
func (StringView) ObjectType() string {
	return astral.String32("").ObjectType()
}

func (v StringView) Render() string {
	return v.Style.Render(v.str.String())
}

// why: str is a pointer and String32.WriteTo is declared on the value receiver, so Go's
// synthesized pointer method dereferences it. A zero-value StringView therefore panicked
// on encode and on decode — the same shape as the NodeInfo nil-identity crash. The
// constructor always sets str; nothing stops a caller composing the zero value.
func (v StringView) WriteTo(writer io.Writer) (n int64, err error) {
	if v.str == nil {
		return astral.String32("").WriteTo(writer)
	}
	return v.str.WriteTo(writer)
}

func (v *StringView) ReadFrom(reader io.Reader) (n int64, err error) {
	if v.str == nil {
		v.str = astral.NewString32("")
	}
	return v.str.ReadFrom(reader)
}

func (v StringView) String() string {
	return v.str.String()
}

func String(text string, style Renderer) *StringView {
	return &StringView{
		Style: style,
		str:   astral.NewString32(text),
	}
}
