# Rules

## Boundary

- astral-go is the wire surface: primitives, wire types, op-name constants,
  protocol clients, client libraries.
- Node-side constructs belong to astrald: `Module` interfaces, engines,
  drivers, authorizer dispatch, `src/` implementations. Never add them here.
- astrald imports astral-go; astral-go never imports astrald.
- A wire-type change is a network-format decision: change the spec in
  `.ai/system` first, then this module, then consumers.
- Shared dependency versions stay pinned to astrald's (btcec, secp256k1).

## Layout

- `astral/` - primitives, codecs, blueprints; `log` nested.
- `sig/`, `streams/` - dependency-free utilities: signal-driven concurrency,
  stream helpers. They import nothing from this module.
- `api/<p>/` - one protocol: wire types + op constants; `client/` is its RPC client.
- `lib/` - app libraries: `apphost` (session), `apps` (serving),
  `astrald` (node client), `ipc`, `query`, `routing`.
- `pub.go` - the root registration aggregator; a blank import of
  `github.com/astralp2p/astral-go` registers the full wire surface.
  Extend it when a new package registers blueprints.

## Wire Types

- Every type defining `ObjectType() string` registers with `astral.Add(&T{})`
  in its defining file.
- Use `astral.Objectify` for `WriteTo`/`ReadFrom`. Objectify fields must be
  astral primitives; plain Go fields are not handled.
- Streaming ops end with `ch.Send(&astral.EOS{})`.
- Send stream errors with `ch.Send(astral.Err(err))`.
- Every `objects.Writer` must `Commit()` or `Discard()`.

## Project APIs

- Prefer `sig.Map`/`sig.Set`/`sig.Queue` over mutex + map/slice.
- Use `sig.RecvErr`/`sig.Recv`/`sig.Send` for context-aware channel ops.
- Logging: always `%v`. Levels: `Log` 0, `Logv(1)` verbose, `Logv(2)` debug.

## Code Shape

- Functions: one responsibility, max 50 lines, max 4 params.
- Packages: one concept. No `util`, `common`, `helpers`.
- Interfaces live at consumers. Prefer 1 method; 3+ is suspect.
- Naming: precise verbs, e.g. `delete`, `find`, `create`.

## Documentation Style

- Write `.ai` docs in the minimal English of `.ai/system/`.
- Declarative present tense. One fact per sentence or bullet.
- No motivation, hype, hedging, or meta-commentary.
- Backtick code identifiers. State defaults, limits, and terminators explicitly.
