package nat

import (
	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// ReflectUDPSession is an open nat.reflect_udp query: the reflector's UDP port
// and token, and the channel that later carries the endpoint it observed.
type ReflectUDPSession struct {
	Port  int
	Token []byte
	ch    *channel.Channel
}

// ReflectUDP opens nat.reflect_udp on the client's target and reads the
// reflector's offer. The caller sends Token to Port from the socket it wants
// reflected, then reads the observed endpoint with Seen.
func (client *Client) ReflectUDP(ctx *astral.Context) (*ReflectUDPSession, error) {
	ch, err := client.queryCh(ctx, nat.MethodReflectUDP, nil)
	if err != nil {
		return nil, err
	}

	var offer *nat.ReflectUDPOffer
	err = ch.Switch(
		func(o *nat.ReflectUDPOffer) error {
			offer = o
			return channel.ErrBreak
		},
		channel.PassErrors,
		channel.WithContext(ctx),
	)
	if err != nil {
		ch.Close()
		return nil, err
	}

	return &ReflectUDPSession{Port: int(offer.Port), Token: offer.Token, ch: ch}, nil
}

// Seen reads the source endpoint the reflector observed for the token.
func (s *ReflectUDPSession) Seen(ctx *astral.Context) (*nat.Endpoint, error) {
	var seen *nat.Endpoint
	err := s.ch.Switch(
		func(e *nat.Endpoint) error {
			seen = e
			return channel.ErrBreak
		},
		channel.PassErrors,
		channel.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	return seen, nil
}

// Close ends the query.
func (s *ReflectUDPSession) Close() error {
	return s.ch.Close()
}
