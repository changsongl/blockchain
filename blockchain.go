package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

const (
	// dbFileNameFormat is a bolt db file name
	dbFileNameFormat = "blockchain_%s.db"

	// blocksBucket is the bucket name of bolt storage
	blocksBucket = "blocks"

	// genesisCoinbaseData is a genesis coinbase data
	genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

	// dbFileMode is database file perm
	dbFileMode = 0600

	// tipDbKey database tip key
	tipDbKey = "l"
)

// Blockchain implements interactions with a DB
type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

// getDBFile returns a bolt database file name
func getDBFile(nodeID string) string {
	return fmt.Sprintf(dbFileNameFormat, nodeID)
}

// CreateBlockchain creates a new blockchain db
func CreateBlockchain(address, nodeID string) *Blockchain {
	dbFileName := getDBFile(nodeID)
	if dbExists(dbFileName) {
		log.Println("blockchain already exists.")
		os.Exit(1)
	}

	cbTx := NewCoinbaseTX(address, genesisCoinbaseData)
	genesisBlock := NewGenesisBlock(cbTx)

	db, err := bolt.Open(dbFileName, dbFileMode, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(createDatabaseFunc(genesisBlock))
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip: genesisBlock.Hash, db: db}
}

// createDatabaseFunc is a function to create a new bolt database
func createDatabaseFunc(genesis *Block) func(tx *bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte(tipDbKey), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}

		return nil
	}
}

// NewBlockchain creates a new Blockchain with genesis Block
func NewBlockchain(nodeID string) *Blockchain {
	dbFileName := getDBFile(nodeID)
	if !dbExists(dbFileName) {
		log.Println("no existing blockchain found, create it first.")
	}

	var tip []byte
	db, err := bolt.Open(dbFileName, dbFileMode, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte(tipDbKey))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip: tip, db: db}
}

// AddBlock saves the block into the blockchain database
func (bc *Blockchain) AddBlock(block *Block) {
	if err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockInDB := b.Get(block.Hash)

		if blockInDB == nil {
			return nil
		}

		blockData := block.Serialize()
		if err := b.Put(block.Hash, blockData); err != nil {
			log.Panic(err)
		}

		lastHash := b.Get([]byte(tipDbKey))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)

		if block.Height > lastBlock.Height {
			if err := b.Put([]byte(tipDbKey), block.Hash); err != nil {
				log.Panic(err)
			}

			bc.tip = block.Hash
		}

		return nil
	}); err != nil {
		log.Panic(err)
	}
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

// dbExists returns whether database file is exists
func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}
