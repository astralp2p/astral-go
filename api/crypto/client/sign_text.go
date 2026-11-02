package crypto

import (
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SignText asks the node to sign text and returns the signature. key selects the
// signer; nil defaults to the caller's own identity key. scheme selects the
// signature scheme; "" defaults to "bip137". The node signs with its own private
// key for key when it holds one, and otherwise through the app that holds a
// crypto.register_signer binding for key. A signature through a binding can wait
// for a person to approve it, for up to the node's signer_timeout (default 5
// minutes).
func (client *Client) SignText(ctx *astral.Context, text string, key *crypto.PublicKey, scheme string) (sig *crypto.Signature, err error) {
	args := query.Args{"text": text}
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

	ch, err := client.queryCh(ctx, crypto.MethodSignText, args)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&sig), channel.PassErrors)
	return
}
