package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// AcceptContract activates a fully-signed node contract as the target node's
// active contract, returning once the node acknowledges it. It is the
// local-setup / cold-card counterpart of AcceptMembership: rather than running
// the signing handshake, the caller supplies a contract already signed by both
// the issuer and the subject, which the node validates, stores, and activates.
//
// The node rejects with code 2 when it already holds an active contract
// (claiming a node is a one-time transition) and streams an error if the
// contract fails validation. Mirrors the astrald op OpAcceptContract.
func (client *Client) AcceptContract(ctx *astral.Context, signed *auth.SignedContract) (err error) {
	ch, err := client.queryCh(ctx, user.OpAcceptContract, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	if err = ch.Send(signed); err != nil {
		return
	}

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}
