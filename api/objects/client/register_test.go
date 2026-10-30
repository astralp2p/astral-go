package objects

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/routing"
)

// stalled accepts a query, reads all the caller sends, and never answers.
func stalled(_ *astral.Context, q *routing.IncomingQuery) error {
	conn := q.AcceptRaw()
	_, _ = io.Copy(io.Discard, conn)
	return conn.Close()
}

// endsWithCtx runs call with a ctx that times out while the node is silent,
// and fails unless call returns that ctx's error soon after.
func endsWithCtx(t *testing.T, call func(*astral.Context) error) {
	t.Helper()
	ctx, cancel := astral.NewContext(nil).WithTimeout(100 * time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- call(ctx) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v; want context.DeadlineExceeded", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("still waiting on the node 5 s after ctx ended")
	}
}

func TestRegisterEndsWithCtx(t *testing.T) {
	c := clientFor(t, objects.MethodRegisterBlueprint, stalled)
	bp := astral.NewBlueprint("test.register",
		astral.Field{Name: "N", Spec: &astral.PrimitiveSpec{PrimitiveType: "uint32"}})

	endsWithCtx(t, func(ctx *astral.Context) error {
		_, err := c.Register(ctx, bp)
		return err
	})
}
