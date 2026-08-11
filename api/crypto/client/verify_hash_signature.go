package crypto

import (
	"encoding/hex"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// VerifyHashSignature asks the node to verify sig against hash. key selects the
// signer to verify against; nil defaults to the caller's own identity key.
// Returns nil when the signature verifies, an error otherwise.
func (client *Client) VerifyHashSignature(ctx *astral.Context, hash []byte, key *crypto.PublicKey, sig *crypto.Signature) (err error) {
	args := query.Args{"hash": hex.EncodeToString(hash)}
	if key != nil {
		text, err := key.MarshalText()
		if err != nil {
			return err
		}
		args["key"] = string(text)
	}

	ch, err := client.queryCh(ctx, crypto.MethodVerifyHashSignature, args)
	if err != nil {
		return
	}
	defer ch.Close()

	if err = ch.Send(sig); err != nil {
		return
	}

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}
