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
// signature scheme; "" defaults to "asn1". The private key never leaves the node.
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
