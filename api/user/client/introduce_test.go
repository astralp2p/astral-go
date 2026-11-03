package user

import (
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/routing"
)

// astrald names the target of user.introduce `identity` and declares it
// required, as it does for user.sync_with (see sync_with_test.go). The op below
// mirrors astrald's declaration and answers with reply, so routing the real
// client through it pins the wire key and the reply handling.

type introduceArgs struct {
	Identity *astral.Identity `query:"required"`
}

// introduceNode records what the op bound and answers reply, standing in for the node.
type introduceNode struct {
	bound chan *astral.Identity
	reply astral.Object
}

func (n *introduceNode) Introduce(_ *astral.Context, q *routing.IncomingQuery, args introduceArgs) error {
	n.bound <- args.Identity

	ch := q.Accept()
	defer ch.Close()

	return ch.Send(n.reply)
}

func clientForIntroduce(t *testing.T, node *introduceNode) *Client {
	t.Helper()

	ops := routing.NewOpRouter()
	op, err := routing.NewOp(node.Introduce)
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	if err = ops.AddOp(user.OpIntroduce, op); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

// The argument the client sends binds to the op's required identity field, and the
// op's ack reaches the caller as a nil error.
func TestIntroduceBindsTheRequiredIdentity(t *testing.T) {
	node := &introduceNode{bound: make(chan *astral.Identity, 1), reply: &astral.Ack{}}
	client := clientForIntroduce(t, node)

	target := astral.GenerateIdentity()

	if err := client.Introduce(astral.NewContext(nil), target); err != nil {
		t.Fatalf("Introduce: %v", err)
	}

	select {
	case got := <-node.bound:
		if got == nil {
			t.Fatal("want the identity bound, got nil: the client named the argument something the op does not declare")
		}
		if !got.IsEqual(target) {
			t.Fatalf("want %s bound, got %s", target, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the op was never called")
	}
}

// A refusal the op sends reaches the caller as an error carrying its message.
func TestIntroducePassesTheRefusal(t *testing.T) {
	node := &introduceNode{bound: make(chan *astral.Identity, 1), reply: user.ErrIntroduceSelf}
	client := clientForIntroduce(t, node)

	err := client.Introduce(astral.NewContext(nil), astral.GenerateIdentity())
	if err == nil {
		t.Fatal("want the refusal, got nil")
	}
	if err.Error() != user.ErrIntroduceSelf.Error() {
		t.Fatalf("want %q, got %q", user.ErrIntroduceSelf, err)
	}
	var msg *astral.ErrorMessage
	if !errors.As(err, &msg) {
		t.Fatalf("want an *astral.ErrorMessage, got %T", err)
	}
}
