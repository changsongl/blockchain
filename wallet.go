package blockchain

import "crypto/ecdsa"

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// HashPubKey hashes public key TODO: implement
func HashPubKey(pubKey []byte) []byte {
	return nil
}