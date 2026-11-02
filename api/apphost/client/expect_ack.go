package apphost

import (
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// expectAck reads an answer that is an ack or an error object. An answer that
// ends without either is an error. The read ends with ctx, and returns ctx's
// error then.
func expectAck(ctx *astral.Context, ch *channel.Channel) error {
	// why FailOnEOF: the node acks only once the operation is done, so an
	// answer that ends without an ack did nothing.
	err := ch.Switch(channel.ExpectAck, channel.PassErrors, channel.FailOnEOF(), channel.WithContext(ctx))

	// why: WithContext ends the read by closing the channel, which reads as an
	// answer cut short, not as the end of ctx.
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
