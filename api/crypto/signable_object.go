package crypto

import "github.com/astralp2p/astral-go/astral"

// SignableObject is a base interface for all contracts
type SignableObject interface {
	astral.Object

	// SignableHash returns the hash of the signable object
	SignableHash() []byte
}

// SignableTextObject is an interface for contracts that can be signed as a text message
type SignableTextObject interface {
	SignableObject

	// SignableText returns a human-readable, printable ASCII line of the object.
	// A text signer signs it after a 23-byte commitment prefix, and a Coldcard
	// signs at most 240 bytes, so the line is at most 217 bytes (see
	// auth.MaxContractTextLen).
	SignableText() string
}
