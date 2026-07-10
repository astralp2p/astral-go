# Examples

Runnable programs, one concept each. Every example builds with the module;
`custom-wire-type` also runs with no node.

The node-backed examples require a running
[astrald](https://github.com/astralp2p/astrald) node and
`ASTRALD_APPHOST_TOKEN` set to an access token; the SDK dials
`tcp:127.0.0.1:8625`.

| Example | Shows | Needs a node |
|---------|-------|--------------|
| [hello-serve](hello-serve/) | serving ops: `lib/apps`, `lib/routing` | yes |
| [hello-query](hello-query/) | sending queries: `lib/astrald`, `lib/query` | yes |
| [objects-store](objects-store/) | the `api/<p>/client` pattern: store an object, read it back | yes |
| [custom-wire-type](custom-wire-type/) | the object model: `astral.Objectify`, `astral.Add`, `astral/channel` | no |

The hello pair talks through the node: both processes authenticate with the
same token, so both resolve to the same guest identity; `hello-serve`
registers a handler for it, and `hello-query` targets it.

```sh
go run ./examples/hello-serve     # terminal 1
go run ./examples/hello-query     # terminal 2, same ASTRALD_APPHOST_TOKEN
```
