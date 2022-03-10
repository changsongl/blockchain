package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

const (
	// TransactionCoinbaseVInVOutDefault is default vout value in first vin for transaction of coinbase
	TransactionCoinbaseVInVOutDefault = -1

	// TransactionCoinbaseVInTxIDDefault is default coinbase transaction id
	TransactionCoinbaseVInTxIDDefault = 0
)

// Transaction represents a transaction
type Transaction struct {
	ID   []byte
	VIn  []TXInput
	VOut []TXOutput
}

// IsCoinbase checks whether the transaction is coinbase
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.VIn) == 1 &&
		len(tx.VIn[0].TxID) == TransactionCoinbaseVInTxIDDefault &&
		tx.VIn[0].VOut == TransactionCoinbaseVInVOutDefault
}

// Serialize returns a serialized Transaction
func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	encoder := gob.NewEncoder(&encoded)
	if err := encoder.Encode(tx); err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// Hash returns hash of the transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

// Sign signs each input of a Transaction
func (tx *Transaction) Sign(privateKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	} else if err := tx.validatePrevTXs(prevTXs); err != nil {
		log.Panic(err)
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.VIn {
		prevTx := prevTXs[hex.EncodeToString(vin.TxID)]
		txCopy.VIn[inID].Signature = nil
		txCopy.VIn[inID].PubKey = prevTx.VOut[vin.VOut].PubKeyHash

		dataToSign := fmt.Sprintf("%x\n", txCopy)

		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, []byte(dataToSign))
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.VIn[inID].Signature = signature
		txCopy.VIn[inID].PubKey = nil
	}
}

// validatePrevTXs validates previous transaction are correct
func (tx *Transaction) validatePrevTXs(prevTXs map[string]Transaction) error {
	for _, vin := range tx.VIn {
		if prevTXs[hex.EncodeToString(vin.TxID)].ID == nil {
			return fmt.Errorf("ERROR: Previous transaction vin.TxID (%s) is not correct", string(vin.TxID))
		}
	}

	return nil
}

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vIn := range tx.VIn {
		inputs = append(inputs, TXInput{TxID: vIn.TxID, VOut: vIn.VOut, Signature: nil, PubKey: nil})
	}

	for _, vOut := range tx.VOut {
		outputs = append(outputs, TXOutput{Value: vOut.Value, PubKeyHash: vOut.PubKeyHash})
	}

	txCopy := Transaction{ID: tx.ID, VIn: inputs, VOut: outputs}

	return txCopy
}
