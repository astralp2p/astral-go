package user

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/routing"
)

// stalledClient returns a client whose target accepts every user op, reads
// all the caller sends, and never answers, as a node does that hangs past the
// accept.
func stalledClient(t *testing.T) *Client {
	t.Helper()
	ops := routing.NewOpRouter()
	for _, name := range []string{
		user.OpAcceptContract, user.OpAcceptMembership, user.OpAddAsset, user.OpAdopt,
		user.OpAssets, user.OpBootstrap, user.OpDeriveIdentity, user.OpExpel, user.OpInfo,
		user.OpJoin, user.OpListExpelled, user.OpListSiblings, user.OpNewNodeContract,
		user.OpRemoveAsset, user.OpRequestMembership, user.OpSwarmStatus, user.OpSyncAssets,
		user.OpSyncWith,
	} {
		op, err := routing.NewOp(func(_ *astral.Context, q *routing.IncomingQuery) error {
			conn := q.AcceptRaw()
			_, _ = io.Copy(io.Discard, conn)
			return conn.Close()
		})
		if err != nil {
			t.Fatalf("NewOp %s: %v", name, err)
		}
		if err := ops.AddOp(name, op); err != nil {
			t.Fatalf("AddOp %s: %v", name, err)
		}
	}
	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

// Every call stops waiting on a target that accepted and never answers when
// its ctx ends, and says that ctx ended rather than handing back an empty
// answer.
func TestEveryCallEndsWithCtx(t *testing.T) {
	c := stalledClient(t)
	node := astral.GenerateIdentity()
	object, _ := astral.ResolveObjectID(astral.NewString8("asset"))
	contract := &auth.Contract{Issuer: node, Subject: node}

	calls := map[string]func(*astral.Context) error{
		"AcceptContract": func(ctx *astral.Context) error {
			return c.AcceptContract(ctx, &auth.SignedContract{Contract: contract})
		},
		"AcceptMembership": func(ctx *astral.Context) error {
			_, err := c.AcceptMembership(ctx, contract, &crypto.Signature{Scheme: "test"})
			return err
		},
		"AddAsset":          func(ctx *astral.Context) error { return c.AddAsset(ctx, object) },
		"Adopt":             func(ctx *astral.Context) error { _, err := c.Adopt(ctx, node.String()); return err },
		"Assets":            func(ctx *astral.Context) error { _, err := c.Assets(ctx); return err },
		"Bootstrap":         func(ctx *astral.Context) error { _, err := c.Bootstrap(ctx, "some words", ""); return err },
		"DeriveIdentity":    func(ctx *astral.Context) error { _, err := c.DeriveIdentity(ctx, "", ""); return err },
		"Expel":             func(ctx *astral.Context) error { _, err := c.Expel(ctx, node); return err },
		"Info":              func(ctx *astral.Context) error { _, err := c.Info(ctx); return err },
		"Join":              func(ctx *astral.Context) error { _, err := c.Join(ctx, node.String()); return err },
		"ListExpelled":      func(ctx *astral.Context) error { _, err := c.ListExpelled(ctx); return err },
		"ListSiblings":      func(ctx *astral.Context) error { _, err := c.ListSiblings(ctx, 0); return err },
		"NewContract":       func(ctx *astral.Context) error { _, err := c.NewContract(ctx, "alias"); return err },
		"RemoveAsset":       func(ctx *astral.Context) error { return c.RemoveAsset(ctx, object) },
		"RequestMembership": func(ctx *astral.Context) error { _, err := c.RequestMembership(ctx); return err },
		"SwarmStatus":       func(ctx *astral.Context) error { _, err := c.SwarmStatus(ctx); return err },
		"SyncAssets":        func(ctx *astral.Context) error { _, _, err := c.SyncAssets(ctx, 0); return err },
		"SyncWith":          func(ctx *astral.Context) error { return c.SyncWith(ctx, node) },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
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
		})
	}
}
