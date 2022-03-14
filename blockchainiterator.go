package blockchain

import (
	"github.com/boltdb/bolt"
	"log"
)

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

// Next returns next block starting from the tip
func (i *BlockchainIterator) Next() *Block {
	var b *Block
	err := i.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(blocksBucket))
		encodeBlock := buc.Get(i.currentHash)
		b = DeserializeBlock(encodeBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = b.PrevBlockHash
	return b
}
