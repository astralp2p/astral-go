package tree

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/tree"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// fakeNode is an in-memory tree.Node: it stores the last Set object and
// creates children on demand.
type fakeNode struct {
	value astral.Object
	subs  map[string]*fakeNode
	sets  int
}

func (n *fakeNode) Get(*astral.Context, bool) (<-chan astral.Object, error) {
	out := make(chan astral.Object, 1)
	if n.value != nil {
		out <- n.value
	}
	close(out)
	return out, nil
}

func (n *fakeNode) Set(_ *astral.Context, object astral.Object) error {
	n.value = object
	n.sets++
	return nil
}

func (n *fakeNode) Delete(*astral.Context) error { return nil }

func (n *fakeNode) Sub(*astral.Context) (map[string]tree.Node, error) {
	out := map[string]tree.Node{}
	for name, sub := range n.subs {
		out[name] = sub
	}
	return out, nil
}

func (n *fakeNode) Create(_ *astral.Context, name string) (tree.Node, error) {
	if n.subs == nil {
		n.subs = map[string]*fakeNode{}
	}
	child := &fakeNode{}
	n.subs[name] = child
	return child, nil
}

func encodeInputs(t *testing.T, objects ...astral.Object) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}
	return &buf
}

func replyTypes(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()
	r := channel.NewReceiver(bytes.NewReader(buf.Bytes()))
	var types []string
	for {
		o, err := r.Receive()
		if errors.Is(err, io.EOF) {
			return types
		}
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		types = append(types, o.ObjectType())
	}
}

// TestSetBatch_ExplicitEOS_MirrorsTerminator: tree.set batch mode stores each
// streamed object and answers an explicit input EOS with a final EOS.
func TestSetBatch_ExplicitEOS_MirrorsTerminator(t *testing.T) {
	a, b := astral.String8("a"), astral.String8("b")
	in := encodeInputs(t, &a, &b, &astral.EOS{})
	var out bytes.Buffer

	root := &fakeNode{}
	ops := NewNodeOps(root)
	ch := channel.New(channel.Join(in, &out))

	err := ops.setBatch(astral.NewContext(nil), ch, SetArgs{Path: "/x"})
	if err != nil {
		t.Fatalf("setBatch: %v", err)
	}

	got := replyTypes(t, &out)
	want := []string{"ack", "ack", "eos"}
	if len(got) != len(want) {
		t.Fatalf("replies %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("replies %v, want %v", got, want)
		}
	}

	leaf := root.subs["x"]
	if leaf == nil || leaf.sets != 2 {
		t.Fatalf("leaf sets = %v, want 2", leaf)
	}
}

// TestSetBatch_EOF_NoTrailingEOS: a stream ended by EOF gets the per-input
// replies and no terminating EOS.
func TestSetBatch_EOF_NoTrailingEOS(t *testing.T) {
	a := astral.String8("a")
	in := encodeInputs(t, &a)
	var out bytes.Buffer

	ops := NewNodeOps(&fakeNode{})
	ch := channel.New(channel.Join(in, &out))

	err := ops.setBatch(astral.NewContext(nil), ch, SetArgs{Path: "/x"})
	if err != nil {
		t.Fatalf("setBatch: %v", err)
	}

	got := replyTypes(t, &out)
	if len(got) != 1 || got[0] != "ack" {
		t.Fatalf("replies %v, want [ack]", got)
	}
}
