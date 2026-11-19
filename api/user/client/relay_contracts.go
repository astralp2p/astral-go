package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// RelayContracts lists the relay contracts the target node holds for issuer:
// active contracts issued by issuer whose every permit is for
// mod.nodes.relay_for_action and that pass the target's admission rule.
// Mirrors the astrald op OpRelayContracts.
//
// The target answers only another member of its local swarm, and refuses
// others with auth.ErrNotPermitted.
func (client *Client) RelayContracts(ctx *astral.Context, issuer *astral.Identity) ([]*auth.SignedContract, error) {
	ch, err := client.queryCh(ctx, user.OpRelayContracts, query.Args{"issuer": issuer})
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var contracts []*auth.SignedContract
	err = ctxErr(ctx, ch.Switch(
		channel.Collect(&contracts),
		channel.BreakOnEOS,
		channel.PassErrors,
		channel.WithContext(ctx),
	))

	return contracts, err
}

// RelayContracts calls the operation on the default client.
func RelayContracts(ctx *astral.Context, issuer *astral.Identity) ([]*auth.SignedContract, error) {
	return Default().RelayContracts(ctx, issuer)
}
