// hello-serve registers with the local astrald node and answers
// "hello?name=alice" queries. Requires a running node; the session
// authenticates with the token in ASTRALD_APPHOST_TOKEN.
package main

import (
	"fmt"

	"github.com/cryptopunkscc/astral-go/astral"
	"github.com/cryptopunkscc/astral-go/lib/apphost"
	"github.com/cryptopunkscc/astral-go/lib/apps"
	"github.com/cryptopunkscc/astral-go/lib/astrald"
	"github.com/cryptopunkscc/astral-go/lib/routing"
)

type API struct{}

type helloArgs struct {
	// tag semantics: "key:<name>;skip;required"; the query param name
	// defaults to the snake_case field name, so this maps to ?name=
	Name string `query:"required"`
}

// Hello becomes the "hello" op via routing.NewApp reflection.
func (api *API) Hello(ctx *astral.Context, q *routing.IncomingQuery, args helloArgs) error {
	ch := q.Accept()
	defer ch.Close()

	return ch.Send(astral.NewString16("hello, " + args.Name))
}

func main() {
	fmt.Printf("hello-serve: serving the hello op; set %s to authenticate with the node\n",
		apphost.AuthTokenEnv)

	// Serve does not return: the context carries no cancellation.
	if err := apps.Serve(astrald.NewContext(), routing.NewApp(&API{})); err != nil {
		panic(err)
	}
}
