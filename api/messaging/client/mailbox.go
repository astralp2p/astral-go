package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Mailbox returns one participant without its credential. id takes a hex
// public key or an alias resolved via the directory.
//
// The node answers "unknown identity" when id resolves to no identity, and
// "identity not found" when its mailbox index has no entry for the identity.
// An entry the node does not serve — one whose hosting contract has expired
// or no longer authorizes — is answered like any other.
func (client *Client) Mailbox(ctx *astral.Context, id string) (*messaging.MailboxInfo, error) {
	ch, err := client.queryCh(ctx, messaging.MethodMailbox, query.Args{"identity": id})
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var info *messaging.MailboxInfo
	if err = ch.Switch(channel.Expect(&info), channel.PassErrors); err != nil {
		return nil, err
	}
	if info == nil {
		return nil, ErrNoAnswer
	}
	return info, nil
}

// Mailbox calls the operation on the default client.
func Mailbox(ctx *astral.Context, id string) (*messaging.MailboxInfo, error) {
	return Default().Mailbox(ctx, id)
}
