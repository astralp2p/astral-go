// hello-query sends a "hello" query to the handler served by hello-serve.
// Requires a running astrald node and the same ASTRALD_APPHOST_TOKEN as hello-serve.
package main

import (
	"fmt"

	"github.com/cryptopunkscc/astral-go/lib/apphost"
	"github.com/cryptopunkscc/astral-go/lib/astrald"
	"github.com/cryptopunkscc/astral-go/lib/query"
)

func main() {
	ctx := astrald.NewContext()

	// hello-serve registers its handler for the guest identity; a nil target
	// routes to the node itself instead.
	guestID := astrald.GuestID()
	if guestID == nil {
		panic("cannot resolve the guest identity: is the node running and " +
			apphost.AuthTokenEnv + " set?")
	}

	ch, err := astrald.WithTarget(guestID).
		QueryChannel(ctx, "hello", query.Args{"name": "alice"})
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	reply, err := ch.Receive()
	if err != nil {
		panic(err)
	}

	fmt.Println(reply)
}
