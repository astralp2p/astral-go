package crypto

import (
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// VerifyTextSignature asks the node to verify sig against text. key selects the
// signer to verify against; nil defaults to the caller's own identity key.
// Returns nil when the signature verifies, an error otherwise.
func (client *Client) VerifyTextSignature(ctx *astral.Context, text string, key *crypto.PublicKey, sig *crypto.Signature) (err error) {
	args := query.Args{"text": text}
	if key != nil {
		keyText, err := key.MarshalText()
		if err != nil {
			return err
		}
		args["key"] = string(keyText)
	}

	ch, err := client.queryCh(ctx, crypto.MethodVerifyTextSignature, args)
	if err != nil {
		return
	}
	defer ch.Close()

	if err = ch.Send(sig); err != nil {
		return
	}

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}
