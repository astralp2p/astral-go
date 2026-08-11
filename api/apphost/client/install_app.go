package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// InstallApp creates, signs, and indexes an app contract for id as a local app
// on the node, and returns the signed contract. id must be an identity whose
// private key the node itself already holds — e.g. one minted by Register —
// since the node signs the contract as both issuer and subject; an external
// identity fails with "sign as issuer: unsupported". Local calls only: the
// node rejects with code 3 when called over the network. duration of 0 uses
// the node's default app-contract duration.
func (client *Client) InstallApp(ctx *astral.Context, id *astral.Identity, duration astral.Duration) (signed *auth.SignedContract, err error) {
	ch, err := client.queryCh(ctx, apphost.MethodInstallApp, query.Args{"id": id, "duration": duration})
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&signed), channel.PassErrors)
	return
}

// InstallApp calls the operation on the default client.
func InstallApp(ctx *astral.Context, id *astral.Identity, duration astral.Duration) (*auth.SignedContract, error) {
	return Default().InstallApp(ctx, id, duration)
}
