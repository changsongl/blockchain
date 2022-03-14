package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/boltdb/bolt"
	"log"
)

// blocksBucket is the bucket name of bolt storage
const blocksBucket = "blocks"

// Blockchain implements interactions with a DB
type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

// Iterator returns a BlockchainIterator
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
}

// FindTransaction finds a transaction by its id
func (bc *Blockchain) FindTransaction(id []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		b := bci.Next()

		for _, tx := range b.Transactions {
			if bytes.Compare(tx.ID, id) == 0 {
				return *tx, nil
			}
		}

		if len(b.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction is not found")
}

// SignTransaction signs inputs of a Transaction
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.VIn {
		prevTx, err := bc.FindTransaction(vin.TxID)
		if err != nil {
			log.Panic(err)
		}

		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(privKey, prevTXs)
}
