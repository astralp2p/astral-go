package services

import (
	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Advertisement is a standing service advertisement. It owns the channel the
// advertisement lives on: the service is available while the Advertisement is
// open and withdrawn once it is closed.
type Advertisement struct {
	ch *channel.Channel
}

// Advertise advertises a named service on the target node. The node takes the
// provider from the caller's identity. info may be nil; SetInfo replaces it
// later.
func (client *Client) Advertise(ctx *astral.Context, name string, info *astral.Bundle) (*Advertisement, error) {
	ch, err := client.queryCh(ctx, services.MethodAdvertise, query.Args{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	return newAdvertisement(ch, info)
}

// newAdvertisement reads the node's answer: an ack becomes a standing
// advertisement, an error object is returned.
func newAdvertisement(ch *channel.Channel, info *astral.Bundle) (*Advertisement, error) {
	if err := ch.Switch(channel.ExpectAck, channel.PassErrors); err != nil {
		ch.Close()
		return nil, err
	}

	ad := &Advertisement{ch: ch}

	if info != nil {
		if err := ad.SetInfo(info); err != nil {
			ad.Close()
			return nil, err
		}
	}

	return ad, nil
}

// Advertise advertises a named service on the default node.
func Advertise(ctx *astral.Context, name string, info *astral.Bundle) (*Advertisement, error) {
	return Default().Advertise(ctx, name, info)
}

// SetInfo replaces the info carried by the advertisement. The service stays
// available across the change.
func (ad *Advertisement) SetInfo(info *astral.Bundle) error {
	return ad.ch.Send(info)
}

// Close withdraws the advertisement.
func (ad *Advertisement) Close() error {
	return ad.ch.Close()
}
