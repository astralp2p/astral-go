package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/sig"
)

// Hosts streams each node that hosts app asynchronously; the error pointer is
// populated only after the channel closes. app is an identity or a name the
// node resolves; a name of no app yields an empty stream.
func (client *Client) Hosts(ctx *astral.Context, app string) (<-chan *astral.Identity, *error) {
	ch, err := client.queryCh(ctx, apphost.MethodHosts, query.Args{"app": app})
	if err != nil {
		return nil, &err
	}

	out := make(chan *astral.Identity)
	errPtr := new(error)

	go func() {
		defer close(out)
		defer ch.Close()

		*errPtr = ch.Switch(
			func(id *astral.Identity) error {
				if id != nil && !id.IsZero() {
					return sig.Send(ctx, out, id)
				}
				return nil
			},
			channel.BreakOnEOS,
			channel.PassErrors,
			channel.WithContext(ctx),
		)
	}()

	return out, errPtr
}

func Hosts(ctx *astral.Context, app string) (<-chan *astral.Identity, *error) {
	return Default().Hosts(ctx, app)
}
