package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
)

// TXOutput represents a transaction outpu
type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	out.PubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// NewTXOutput creates a new TXOutput
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{Value: value, PubKeyHash: nil}
	txo.Lock([]byte(address))

	return txo
}

// TXOutputs collects TXOutput
type TXOutputs struct {
	Outputs []TXOutput
}

// DeserializeOutputs deserializes byte slice to TXOutputs
func DeserializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&outputs); err != nil {
		log.Panic(err)
	}

	return outputs
}

// Serialize serializes TXOutputs
func (outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(outs); err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
