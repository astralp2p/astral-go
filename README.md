# astral-go

The Go SDK of the Astral Network: the astral object model, the protocol wire
types and clients under `api/`, and the app libraries under `lib/`.

## Install

```sh
go get github.com/cryptopunkscc/astral-go
```

Requires Go 1.25. Talking to the network requires a running
[astrald](https://github.com/cryptopunkscc/astrald) node: the SDK dials
`tcp:127.0.0.1:8625` and reads the access token from
`ASTRALD_APPHOST_TOKEN`.

## Send a query

```go
package main

import (
	"fmt"

	"github.com/cryptopunkscc/astral-go/lib/astrald"
	"github.com/cryptopunkscc/astral-go/lib/query"
)

func main() {
	ctx := astrald.NewContext()

	// routes "hello?name=alice" to the local node
	ch, err := astrald.QueryChannel(ctx, "hello", query.Args{"name": "alice"})
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
```

## Serve queries

```go
package main

import (
	"github.com/cryptopunkscc/astral-go/astral"
	"github.com/cryptopunkscc/astral-go/lib/apps"
	"github.com/cryptopunkscc/astral-go/lib/astrald"
	"github.com/cryptopunkscc/astral-go/lib/routing"
)

type API struct{}

type helloArgs struct {
	Name string `query:"required"`
}

// Hello answers queries of the form "hello?name=alice".
func (api *API) Hello(ctx *astral.Context, q *routing.IncomingQuery, args helloArgs) error {
	ch := q.Accept()
	defer ch.Close()

	return ch.Send(astral.NewString16("hello, " + args.Name))
}

func main() {
	if err := apps.Serve(astrald.NewContext(), routing.NewApp(&API{})); err != nil {
		panic(err)
	}
}
```

`routing.NewApp` reflects exported methods with the signature
`func(*astral.Context, *routing.IncomingQuery[, ArgsStruct]) error` into
snake_case ops and mounts a `.spec` manifest op. `apps.Serve` opens a local
IPC listener, keeps the app registered with the node across reconnects, and
blocks until the context ends.

## Examples

Runnable programs under [`examples/`](examples/), one concept each:

- `hello-serve` / `hello-query` — an op-serving app and its caller; run both
  with the same `ASTRALD_APPHOST_TOKEN`.
- `objects-store` — store an object, read it back by ID.
- `custom-wire-type` — define, register, and exchange a wire type over an
  in-process pipe; runs without a node.

## Layout

- `astral/` — the object model: `Object`, `Identity`, `ObjectID`,
  `Blueprints`, `Query`, `Context`; subpackages `channel` (typed object
  streams over any bytestream), `fmt`, `log`.
- `api/<p>/` — one package per protocol: wire types and op-name constants;
  `client/`, where present, is the protocol's RPC client.
- `lib/` — app libraries: `astrald` (the outbound client), `apps` (serving),
  `apphost` (the node session under `astrald`), `routing` (op dispatch),
  `query` (query strings), `ipc` (local transport).
- `sig/`, `streams/` — dependency-free utilities: signal-driven concurrency,
  stream helpers.
- `pub.go` — the registration aggregator: a blank import of
  `github.com/cryptopunkscc/astral-go` registers the astral primitives and
  every `api/` wire type, so `astral.Decode` materializes them by type name.

## The spec

Protocol truth lives in
[astral-docs](https://github.com/cryptopunkscc/astral-docs), pinned at
`.ai/system`:

```sh
git submodule update --init
```

- `.ai/system/core-definitions/` — Identity, Node, Query, Channel, Object
- `.ai/system/protocols/<p>/` — per-protocol op and type specs
- `.ai/system/topics/` — encodings, the IPC and transport protocols

`.ai/` is the AI workspace; an agent working in this repo loads
`.ai/README.md` first.

## Boundary

Apps import astral-go; astrald imports astral-go; astral-go imports neither.
What runs the network belongs to astrald; what talks to the network belongs
here.
