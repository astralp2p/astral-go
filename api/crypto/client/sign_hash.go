package crypto

import (
	"encoding/hex"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SignHash asks the node to sign hash and returns the signature. key selects the
// signer; nil defaults to the caller's own identity key. scheme selects the
// signature scheme; "" defaults to "asn1". The node signs with its own private
// key for key first, and otherwise through the app that holds a
// crypto.register_signer binding for key. A signature through a binding can wait
// for a person to approve it, for up to the node's signer_timeout (default 5
// minutes).
func (client *Client) SignHash(ctx *astral.Context, hash []byte, key *crypto.PublicKey, scheme string) (sig *crypto.Signature, err error) {
	args := query.Args{"hash": hex.EncodeToString(hash)}
	if scheme != "" {
		args["scheme"] = scheme
	}
	if key != nil {
		text, err := key.MarshalText()
		if err != nil {
			return nil, err
		}
		args["key"] = string(text)
	}

	ch, err := client.queryCh(ctx, crypto.MethodSignHash, args)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&sig), channel.PassErrors)
	return
}
