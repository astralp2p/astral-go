package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Wait parks on the caller's inbox until it holds a message the caller has not
// archived, or the granted window closes. It stamps nothing: the park and the
// read are separate acts.
//
// Cancelling ctx closes the channel, which ends the park on the node.
func (client *Client) Wait(ctx *astral.Context, req messaging.WaitRequest) (*messaging.WaitResult, error) {
	args := query.Args{}
	if req.From != "" {
		args["from"] = req.From
	}
	if req.Since != 0 {
		args["since"] = req.Since
	}
	if req.Timeout != 0 {
		args["timeout"] = astral.Duration(req.Timeout)
	}

	ch, err := client.queryCh(ctx, messaging.MethodWait, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var res *messaging.WaitResult
	err = ch.Switch(channel.Expect(&res), channel.PassErrors, channel.WithContext(ctx))
	if err == nil && res != nil {
		return res, nil
	}

	// why the context's error first: a cancelled park closes the channel, and
	// the read error or missing answer that follows is the caller's own doing.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	return nil, errNoAnswer
}

// Wait calls the operation on the default client.
func Wait(ctx *astral.Context, req messaging.WaitRequest) (*messaging.WaitResult, error) {
	return Default().Wait(ctx, req)
}
