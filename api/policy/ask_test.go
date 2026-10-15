package policy

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// delegateNode stands in for the node routing the question to a delegate. It
// records the operation and the request, then answers with answer, hangs up
// when hangUp is set, or holds the question open otherwise.
type delegateNode struct {
	id     *astral.Identity
	answer astral.Object
	hangUp bool

	queried  chan string
	received chan astral.Object
	closed   chan struct{}
}

func newDelegateNode(answer astral.Object) *delegateNode {
	return &delegateNode{
		id:       astral.GenerateIdentity(),
		answer:   answer,
		queried:  make(chan string, 1),
		received: make(chan astral.Object, 1),
		closed:   make(chan struct{}),
	}
}

func (n *delegateNode) Identity() *astral.Identity { return n.id }

func (n *delegateNode) RouteQuery(_ *astral.Context, q *astral.InFlightQuery, w io.WriteCloser) (io.WriteCloser, error) {
	n.queried <- string(q.QueryString)

	return query.Accept(q, w, func(conn astral.Conn) {
		defer close(n.closed)
		// why: an op's connection closes when the op returns, so a delegate
		// that returns without answering hangs up on the asker.
		defer conn.Close()
		ch := channel.New(conn)

		obj, err := ch.Receive()
		if err != nil {
			return
		}
		n.received <- obj

		switch {
		case n.hangUp:
			return
		case n.answer == nil:
			_, _ = io.Copy(io.Discard, conn)
			return
		}
		_ = ch.Send(n.answer)
	})
}

// unroutable stands in for a node that cannot reach the delegate.
type unroutable struct{ id *astral.Identity }

func (n unroutable) Identity() *astral.Identity { return n.id }

func (unroutable) RouteQuery(*astral.Context, *astral.InFlightQuery, io.WriteCloser) (io.WriteCloser, error) {
	return nil, errors.New("no route")
}

func testDelegate() *Delegate {
	return &Delegate{Identity: astral.GenerateIdentity(), Operation: "control.decide"}
}

// The delegate is asked at its operation with the request, and its answer is
// returned as given.
func TestAskReturnsTheDelegatesAnswer(t *testing.T) {
	yes := astral.Bool(true)
	node := newDelegateNode(&yes)

	obj, err := Ask(astral.NewContext(nil), node, testDelegate(), &astral.Ack{})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if op := <-node.queried; op != "control.decide" {
		t.Fatalf("asked %q, want the configured operation", op)
	}
	if _, ok := (<-node.received).(*astral.Ack); !ok {
		t.Fatal("the request did not reach the delegate")
	}
	if got, ok := obj.(*astral.Bool); !ok || !bool(*got) {
		t.Fatalf("answer %T %v, want the delegate's Bool true", obj, obj)
	}
}

// A delegate that hangs up without answering has decided nothing.
func TestAskFailsWhenTheDelegateAnswersNothing(t *testing.T) {
	node := newDelegateNode(nil)
	node.hangUp = true

	if _, err := Ask(astral.NewContext(nil), node, testDelegate(), &astral.Ack{}); err == nil {
		t.Fatal("an empty answer was returned as a decision")
	}
}

// A question that cannot be put is an error, not an answer.
func TestAskFailsWhenTheDelegateIsUnreachable(t *testing.T) {
	node := unroutable{id: astral.GenerateIdentity()}

	if _, err := Ask(astral.NewContext(nil), node, testDelegate(), &astral.Ack{}); err == nil {
		t.Fatal("an unreachable delegate answered")
	}
}

// A delegate holding the question does not hold the asker once ctx ends: the
// ask fails and the question closes at the delegate.
func TestAskEndsWithItsContext(t *testing.T) {
	node := newDelegateNode(nil)
	ctx, cancel := astral.NewContext(nil).WithCancel()

	done := make(chan error, 1)
	go func() {
		_, err := Ask(ctx, node, testDelegate(), &astral.Ack{})
		done <- err
	}()

	<-node.received
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a cancelled question returned an answer")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the ask kept waiting after ctx ended")
	}

	select {
	case <-node.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("the question stayed open at the delegate")
	}
}

// The delegate's timeout bounds the wait on its own.
func TestAskEndsAtTheDelegatesTimeout(t *testing.T) {
	node := newDelegateNode(nil)
	d := testDelegate()
	d.Timeout = astral.Duration(50 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := Ask(astral.NewContext(nil), node, d, &astral.Ack{})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a timed-out question returned an answer")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the ask kept waiting past the delegate's timeout")
	}

	select {
	case <-node.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("the question stayed open at the delegate")
	}
}
