package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
	"log"
)

// version
const version = byte(0x00)

// addressChecksumLen is the checking length for address
const addressChecksumLen = 4

// Wallet stores private and public keys
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet creates and returns a new wallet
func NewWallet() *Wallet {
	private, public := newKeyPair()
	wallet := &Wallet{PrivateKey: private, PublicKey: public}

	return wallet
}

// HashPubKey hashes public key
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	ripemd160Hasher := ripemd160.New()
	if _, err := ripemd160Hasher.Write(publicSHA256[:]); err != nil {
		log.Panic(err)
	}
	publicRIPEME160 := ripemd160Hasher.Sum(nil)

	return publicRIPEME160
}

// GetAddress returns wallet address
func (w Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte(version), pubKeyHash...)
	checkSum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checkSum...)
	address := Base58Encode(fullPayload)

	return address
}

// ValidateAddress check if address if valid
func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	ver := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{ver}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

// checksum generates a check sum for a public key
func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

// newKeyPair creates a new pair of public and private keys
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}
