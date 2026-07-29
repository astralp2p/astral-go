package routing

import (
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// App is a ScopeRouter that describes itself: NewApp mounts a struct's ops and
// injects a ".spec" op streaming the manifest, so a caller can discover an app
// without out-of-band documentation.

type appOps struct{}

func (appOps) Ping(_ *astral.Context, q *IncomingQuery) error { return q.Reject() }
func (appOps) Read(_ *astral.Context, q *IncomingQuery, _ opArgs) error {
	return q.Reject()
}

type scopeOps struct{}

func (scopeOps) List(_ *astral.Context, q *IncomingQuery) error { return q.Reject() }

func TestNewApp_MountsTheStructOps(t *testing.T) {
	app := NewApp(&appOps{})

	for _, name := range []string{"ping", "read"} {
		if !app.HasRoute(name) {
			t.Fatalf("want %q mounted", name)
		}
	}
}

// The ".spec" op is injected, not declared by the app author.
func TestNewApp_InjectsTheSpecOp(t *testing.T) {
	app := NewApp(&appOps{})

	if !app.HasRoute(".spec") {
		t.Fatal("want .spec mounted")
	}
}

// The manifest lists the app's own ops and withholds the injected ".spec".
//
// Note that App.Spec is the op handler, which shadows ScopeRouter.Spec; the
// manifest is reached through the embedded router by name.
func TestApp_SpecManifestExcludesTheSpecOp(t *testing.T) {
	app := NewApp(&appOps{})

	names := map[string]bool{}
	for _, spec := range app.ScopeRouter.Spec() {
		names[spec.Name] = true
	}

	if !names["ping"] || !names["read"] {
		t.Fatalf("want the app ops listed, got %v", names)
	}
	if names[".spec"] {
		t.Fatalf("want .spec withheld from the manifest, got %v", names)
	}
}

func TestApp_AddMountsAScope(t *testing.T) {
	app := NewApp(&appOps{})
	app.Add("objects", &scopeOps{})

	if !app.HasRoute("objects.list") {
		t.Fatal("want the scoped op mounted")
	}

	names := map[string]bool{}
	for _, spec := range app.ScopeRouter.Spec() {
		names[spec.Name] = true
	}
	if !names["objects.list"] {
		t.Fatalf("want the scoped op in the manifest, got %v", names)
	}
}

// The ".spec" op streams one OpSpec per operation and terminates with EOS, so
// a caller knows the manifest is complete.
func TestApp_SpecOpStreamsTheManifestThenEOS(t *testing.T) {
	app := NewApp(&appOps{})

	id := astral.GenerateIdentity()
	ctx := astral.NewContext(nil).WithIdentity(id)

	ch, err := query.Route(ctx, app, query.New(id, id, ".spec", nil))
	if err != nil {
		t.Fatalf("Route: unexpected error: %v", err)
	}
	defer ch.Close()

	type result struct {
		names []string
		eos   bool
		err   error
	}
	done := make(chan result, 1)

	go func() {
		var r result
		for {
			obj, err := ch.Receive()
			if err != nil {
				r.err = err
				done <- r
				return
			}

			switch o := obj.(type) {
			case *OpSpec:
				r.names = append(r.names, o.Name)
			case *astral.EOS:
				r.eos = true
				done <- r
				return
			}
		}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("Receive: unexpected error: %v", r.err)
		}
		if !r.eos {
			t.Fatal("want the stream terminated with EOS")
		}

		got := map[string]bool{}
		for _, name := range r.names {
			got[name] = true
		}
		if !got["ping"] || !got["read"] {
			t.Fatalf("want the app ops streamed, got %v", r.names)
		}
		if got[".spec"] {
			t.Fatalf("want .spec withheld, got %v", r.names)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the spec stream did not complete")
	}
}

// Run needs an op name; without one it reports the omission rather than
// dialling an empty query.
func TestApp_RunRequiresACommand(t *testing.T) {
	app := NewApp(&appOps{})

	err := app.Run(astral.NewContext(nil), nil)
	if err == nil {
		t.Fatal("want an error for missing arguments, got nil")
	}

	err = app.Run(astral.NewContext(nil), []string{})
	if err == nil {
		t.Fatal("want an error for empty arguments, got nil")
	}
}

// App.Run beyond the empty-argument branch binds os.Stdin and os.Stdout
// directly, so it is not exercised here.
