/*
Package crypto provides a module with cryptographic operations and objects.
*/
package crypto

const (
	MethodPublicKey           = "crypto.public_key"
	MethodRegisterSigner      = "crypto.register_signer"
	MethodSignHash            = "crypto.sign_hash"
	MethodSignText            = "crypto.sign_text"
	MethodVerifyHashSignature = "crypto.verify_hash_signature"
	MethodVerifyTextSignature = "crypto.verify_text_signature"
)
