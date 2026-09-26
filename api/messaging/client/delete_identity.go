package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// DeleteIdentity removes a participant: revokes every access token of it, its
// grants and its alias, and stops hosting its mailbox, deleting the mail it
// owns. It revokes no contract: the signed relay and hosting contracts stay
// valid until they expire, and the withdrawal holds on this node alone. id
// takes a hex public key or an alias resolved via the directory, and the errors
// are Identity's.
func (client *Client) DeleteIdentity(ctx *astral.Context, id string) error {
	ch, err := client.queryCh(ctx, messaging.MethodDeleteIdentity, query.Args{"identity": id})
	if err != nil {
		return err
	}
	defer ch.Close()

	// why not channel.ExpectAck: Switch returns nil both on the ack and on a
	// query the node closed without one, and the latter is no deletion.
	var ack *astral.Ack
	if err = ch.Switch(channel.Expect(&ack), channel.PassErrors); err != nil {
		return err
	}
	if ack == nil {
		return ErrNoAnswer
	}
	return nil
}

// DeleteIdentity calls the operation on the default client.
func DeleteIdentity(ctx *astral.Context, id string) error {
	return Default().DeleteIdentity(ctx, id)
}
