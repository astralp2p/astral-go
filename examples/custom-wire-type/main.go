// custom-wire-type defines a Note wire type and exchanges it over an in-memory
// pipe. No astrald node is involved.
package main

import (
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/streams"
)

var _ astral.Object = &Note{}

// Prefer astral primitives for wire fields; Objectify rejects platform-width
// int/uint (use sized types).
type Note struct {
	Author astral.String8
	Text   astral.String16
}

func (n Note) ObjectType() string {
	return "example.note"
}

func (n Note) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&n).WriteTo(w)
}

func (n *Note) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(n).ReadFrom(r)
}

func init() {
	_ = astral.Add(&Note{})
}

func main() {
	left, right := streams.Pipe()

	go func() {
		ch := channel.New(left)
		defer ch.Close()
		if err := ch.Send(&Note{Author: "alice", Text: "hello over the wire"}); err != nil {
			panic(err)
		}
		if err := ch.Send(&astral.EOS{}); err != nil {
			panic(err)
		}
	}()

	ch := channel.New(right)
	defer ch.Close()
	for {
		obj, err := ch.Receive()
		if err != nil {
			panic(err)
		}

		switch o := obj.(type) {
		case *Note:
			fmt.Printf("note from %v: %v\n", o.Author, o.Text)
		case *astral.EOS:
			return
		}
	}
}
