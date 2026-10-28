package apphost

import (
	"fmt"
	"strings"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Register provisions a fresh guest identity on the default node.
func Register(ctx *astral.Context, permits ...string) (*apphost.AccessToken, error) {
	return Default().Register(ctx, permits...)
}

// RegisterWith provisions a fresh guest identity on the default node.
func RegisterWith(ctx *astral.Context, opts RegisterOptions) (*apphost.AccessToken, error) {
	return Default().RegisterWith(ctx, opts)
}

// RegisterOptions is what an app asks apphost.register for. All fields are
// optional; an empty field sends no argument.
type RegisterOptions struct {
	// Name is the name the app gives itself, shown to a delegated register
	// policy. The node does not verify it.
	Name string

	// GrantPermits are actions asked for as node-local grants, e.g.
	// `mod.user.see_swarm_action`: revocable by deleting a row, and worthless
	// off this node.
	GrantPermits []string

	// ContractPermits are actions asked for in a signed node→app contract:
	// portable evidence another node verifies, durable until it expires.
	ContractPermits []string
}

// Register provisions a fresh guest identity end to end: the node generates a
// keypair, signs an app contract between the new identity and itself, and
// mints an access token for it. This is how an app bootstraps itself on first
// run, so the session issuing the query is normally anonymous — the credential
// being asked for is the one the caller does not have yet.
//
// permits names the actions the new identity asks to hold as node-local
// grants, e.g. `mod.user.see_swarm_action`. A grant is revocable by deleting
// its row and is worthless off this node. Asking is not receiving: the node's
// register policy decides what it grants, and the answer carries the token
// alone, so a caller learns what it holds by using it. Asking for nothing
// sends no argument.
//
// Register asks for grants alone, so nothing it names becomes portable.
// RegisterWith also names the app and asks for contract permits.
//
// The node accepts the query before its register policy decides, so a refusal
// arrives as the error object apphost.ErrRegistrationDeclined, which surfaces
// here as an error, as does any failure past the accept gate.
func (client *Client) Register(ctx *astral.Context, permits ...string) (*apphost.AccessToken, error) {
	return client.RegisterWith(ctx, RegisterOptions{GrantPermits: permits})
}

// RegisterWith is Register with the app's name and both permit lists. A
// permit name carrying a comma is refused with apphost.ErrProtocolError before
// anything is sent.
//
// why a second method: Register's variadic permits leave no room for options,
// and existing apps call it as it is.
func (client *Client) RegisterWith(ctx *astral.Context, opts RegisterOptions) (*apphost.AccessToken, error) {
	args, err := registerArgs(opts)
	if err != nil {
		return nil, err
	}

	ch, err := client.queryCh(ctx, apphost.MethodRegister, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	return readAccessToken(ch)
}

// The arguments apphost.register reads. The operation declares GrantPermits,
// ContractPermits and Name, and the op router lowers a field name to
// snake_case, so these are the names on the wire (astrald
// `mod/apphost/src/op_register.go`, `opRegisterArgs`).
//
// The names are pinned by tests because getting one wrong is silent: an
// argument the operation does not declare is skipped during binding, so a
// client naming the wrong one registers successfully and holds nothing it
// asked for.
const (
	argGrantPermits    = "grant_permits"
	argContractPermits = "contract_permits"
	argName            = "name"
)

// registerArgs builds the op's arguments from opts, leaving out every empty
// field.
func registerArgs(opts RegisterOptions) (query.Args, error) {
	args := query.Args{}
	if opts.Name != "" {
		args[argName] = opts.Name
	}
	if err := addPermits(args, argGrantPermits, opts.GrantPermits); err != nil {
		return nil, err
	}
	if err := addPermits(args, argContractPermits, opts.ContractPermits); err != nil {
		return nil, err
	}
	return args, nil
}

// addPermits joins permits into args[key]. A permit carrying the separator is
// refused rather than sent: on the wire it would split into two, so the node
// would read an action name the caller never asked for.
func addPermits(args query.Args, key string, permits []string) error {
	if len(permits) == 0 {
		return nil
	}

	for _, permit := range permits {
		if strings.Contains(permit, ",") {
			return fmt.Errorf("%w: permit name contains a comma: %q", apphost.ErrProtocolError, permit)
		}
	}

	args[key] = strings.Join(permits, ",")
	return nil
}

// readAccessToken reads the node's answer: the minted token, or the error
// object a failed step streams. An answer carrying neither is a protocol
// error — registration has no empty success, and a nil token returned with a
// nil error would strand the caller at the one call it cannot retry blind.
func readAccessToken(ch *channel.Channel) (token *apphost.AccessToken, err error) {
	if err = ch.Switch(channel.Expect(&token), channel.PassErrors); err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("%w: apphost.register returned no access token", apphost.ErrProtocolError)
	}

	return token, nil
}
