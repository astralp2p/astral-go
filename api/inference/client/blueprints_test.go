package inference

import (
	"errors"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/routing"
)

// registry stands in for a node's blueprints: the primitives, and no
// inference type until one is registered.
func registry(t *testing.T) *astral.Blueprints {
	t.Helper()
	reg := astral.NewBlueprints(nil)
	for _, name := range []string{"string8", "string16", "string32", "bytes32", "uint32", "bool", "float32", "object_id.sha256"} {
		if err := reg.Add(astral.New(name)); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
	}
	return reg
}

// blueprintNode answers objects.register_blueprint and objects.get_blueprint
// from reg, the way astrald does.
func blueprintNode(t *testing.T, reg *astral.Blueprints) (*Client, *astral.Identity) {
	t.Helper()
	return clientFor(t, map[string]any{
		objects.MethodRegisterBlueprint: func(_ *astral.Context, q *routing.IncomingQuery, _ struct{ In, Out string }) error {
			ch := q.Accept()
			defer ch.Close()
			return channel.Batch(ch, func(bp *astral.Blueprint) astral.Object {
				id, err := reg.RegisterBlueprint(bp)
				if err != nil {
					return astral.NewError(err.Error())
				}
				return id
			})
		},
		objects.MethodGetBlueprint: func(_ *astral.Context, q *routing.IncomingQuery, args struct {
			Type string `query:"required"`
			Out  string
		}) error {
			ch := q.Accept()
			defer ch.Close()
			if bp := reg.GetBlueprint(args.Type); bp != nil {
				return ch.Send(bp)
			}
			return ch.Send(astral.NewError("blueprint not found: " + args.Type))
		},
	})
}

// The push registers every type on a node that knows none, dependencies
// first, and pushing again is a success.
func TestPushBlueprints(t *testing.T) {
	reg := registry(t)
	c, _ := blueprintNode(t, reg)
	ctx := astral.NewContext(nil)

	if err := c.PushBlueprints(ctx); err != nil {
		t.Fatal(err)
	}
	for _, o := range inference.Objects() {
		if reg.GetBlueprint(o.ObjectType()) == nil {
			t.Errorf("the node lacks %s", o.ObjectType())
		}
	}
	if err := c.PushBlueprints(ctx); err != nil {
		t.Fatalf("a second push: %v", err)
	}
	if err := c.CheckBlueprints(ctx); err != nil {
		t.Fatalf("check after a push: %v", err)
	}
}

func TestPushBlueprintsFails(t *testing.T) {
	c, _ := clientFor(t, map[string]any{
		objects.MethodRegisterBlueprint: func(_ *astral.Context, q *routing.IncomingQuery) error {
			return q.Reject()
		},
	})
	if err := c.PushBlueprints(astral.NewContext(nil)); err == nil {
		t.Fatal("a push the node rejects succeeded")
	}
}

// A node that accepts a push and stalls holds the caller only until ctx ends.
func TestPushBlueprintsEndsWithCtx(t *testing.T) {
	c, _ := clientFor(t, map[string]any{objects.MethodRegisterBlueprint: stalled})
	endsWithCtx(t, c.PushBlueprints)
}

// The check names the end of ctx, not a broken answer, when the node stalls.
func TestCheckBlueprintsEndsWithCtx(t *testing.T) {
	c, _ := clientFor(t, map[string]any{objects.MethodGetBlueprint: stalled})
	endsWithCtx(t, c.CheckBlueprints)
}

// A node that loaded the v1 inference.chunk since it started answers the push
// of v2 with "already registered", and only the check finds it out.
func TestCheckBlueprintsFindsAnOlderShape(t *testing.T) {
	reg := registry(t)
	v1 := astral.NewBlueprint("inference.chunk",
		astral.Field{Name: "Index", Spec: &astral.PrimitiveSpec{PrimitiveType: "uint32"}},
		astral.Field{Name: "Text", Spec: &astral.PrimitiveSpec{PrimitiveType: "string16"}},
		astral.Field{Name: "Reasoning", Spec: &astral.PrimitiveSpec{PrimitiveType: "string16"}},
		astral.Field{Name: "FinishReason", Spec: &astral.PrimitiveSpec{PrimitiveType: "string8"}},
		astral.Field{Name: "Model", Spec: &astral.PrimitiveSpec{PrimitiveType: "string8"}},
	)
	if _, err := reg.RegisterBlueprint(v1); err != nil {
		t.Fatal(err)
	}
	c, id := blueprintNode(t, reg)
	ctx := astral.NewContext(nil)

	if err := c.PushBlueprints(ctx); err != nil {
		t.Fatalf("push: %v", err)
	}
	err := c.CheckBlueprints(ctx)
	var stale *StaleBlueprintError
	if !errors.As(err, &stale) || stale.Type != "inference.chunk" || !stale.Node.IsEqual(id) {
		t.Fatalf("err = %v; want a stale inference.chunk on %v", err, id)
	}
	if want := "node " + id.String() + " holds an older inference.chunk; restart the node"; err.Error() != want {
		t.Fatalf("message %q; want %q", err.Error(), want)
	}
}

// A type the node lacks is an error of its own, not a stale shape.
func TestCheckBlueprintsMissingType(t *testing.T) {
	c, _ := blueprintNode(t, registry(t))

	err := c.CheckBlueprints(astral.NewContext(nil))
	var stale *StaleBlueprintError
	if err == nil || errors.As(err, &stale) || !strings.Contains(err.Error(), "inference.effort_level") {
		t.Fatalf("err = %v; want the missing inference.effort_level", err)
	}
}
