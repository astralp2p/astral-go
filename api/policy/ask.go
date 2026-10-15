package policy

import (
	"context"
	"fmt"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// Ask puts req to d as node and returns the one object d answers. The wait
// ends at the earliest of the answer, the end of ctx, and d's bound.
//
// An error is a question that could not be put, or an answer that did not
// arrive; neither is a decision, and a caller refuses on it. The caller reads
// the answer's type: an object other than its decision type is an answer it
// cannot read, and an unreadable answer must not read as permission.
func Ask(ctx *astral.Context, node astral.Node, d *Delegate, req astral.Object) (astral.Object, error) {
	ctx, cancel := ctx.WithTimeout(d.timeout())
	defer cancel()

	ch, err := query.Route(ctx, node, query.New(node.Identity(), d.Identity, string(d.Operation), nil))
	if err != nil {
		return nil, fmt.Errorf("delegate %v: %w", d.Identity, err)
	}
	defer ch.Close()

	// why: Receive does not watch ctx, so the end of ctx closes the channel to
	// end the read.
	defer context.AfterFunc(ctx, func() { _ = ch.Close() })()

	if err = ch.Send(req); err != nil {
		return nil, fmt.Errorf("delegate %v: %w", d.Identity, err)
	}

	obj, err := ch.Receive()
	switch {
	case err != nil:
		return nil, fmt.Errorf("delegate %v: %w", d.Identity, err)
	case obj == nil:
		return nil, fmt.Errorf("delegate %v answered nothing", d.Identity)
	}

	return obj, nil
}
