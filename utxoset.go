package blockchain

import (
	"encoding/hex"
	"github.com/boltdb/bolt"
	"log"
)

// utxoBucket is unspent transaction bucket name
const utxoBucket = "chainstate"

// UTXOSet represents UTXO set
type UTXOSet struct {
	Blockchain *Blockchain
}

// NewUTXOSet creates and returns a UTXOSet
func NewUTXOSet(bc *Blockchain) UTXOSet {
	return UTXOSet{Blockchain: bc}
}

// FindSpendableOutputs finds and returns unspent outputs to reference in inputs
func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.db

	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}

		return nil
	}); err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs
}

// Reindex rebuilds the UTXO set
func (u UTXOSet) Reindex() {
	bucket := []byte(utxoBucket)

	if err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(bucket); err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		if _, err := tx.CreateBucket(bucket); err != nil {
			log.Panic(err)
		}

		return nil
	}); err != nil {
		log.Panic(err)
	}

	utxo := u.Blockchain.FindUTXO()
	if err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)

		for txID, outs := range utxo {
			key, err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}

			if err = b.Put(key, outs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	}); err != nil {
		log.Panic(err)
	}
}
