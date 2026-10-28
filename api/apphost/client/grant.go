package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// GrantOptions configures apphost.grant. All fields are optional.
type GrantOptions struct {
	// Duration is the grant's lifetime. Zero grants until the grant is
	// revoked.
	Duration astral.Duration

	// Constraints narrow the granted permit; the action decides what they
	// mean. mod.services.service_discovery_action covers nothing
	// unconstrained, so its grant carries a bundle holding one
	// services.DiscoveryScope. Nil grants the action without constraints.
	Constraints *astral.Bundle
}

// Grant records a node-local grant of action for identity on the target node,
// replacing whatever identity held for that action, so a narrower grant takes
// authority away. The grant authorizes on that node alone and travels nowhere.
//
// The caller must hold mod.auth.admin_manage_apps_action. The node rejects
// any other caller, and any query that arrived over a link, before it answers.
// A failure past the accept gate, such as an identity that does not resolve,
// arrives as an error, and so does an answer that ends without an ack.
func (client *Client) Grant(ctx *astral.Context, identity *astral.Identity, action string, opts GrantOptions) error {
	args := query.Args{"identity": identity.String(), "action": action}
	if opts.Duration != 0 {
		args["duration"] = opts.Duration
	}
	if opts.Constraints != nil {
		args["constrained"] = true
	}

	ch, err := client.queryCh(ctx, apphost.MethodGrant, args)
	if err != nil {
		return err
	}
	defer ch.Close()

	// why: the constraints travel on the channel because a bundle is not a
	// query argument; the node reads exactly one before it grants.
	if opts.Constraints != nil {
		if err = ch.Send(opts.Constraints); err != nil {
			return err
		}
	}

	// why FailOnEOF: the node acks only once the grant is written, so an
	// answer that ends without an ack recorded nothing.
	return ch.Switch(channel.ExpectAck, channel.PassErrors, channel.FailOnEOF(), channel.WithContext(ctx))
}

// Grant calls the operation on the default client.
func Grant(ctx *astral.Context, identity *astral.Identity, action string, opts GrantOptions) error {
	return Default().Grant(ctx, identity, action, opts)
}
