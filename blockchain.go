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

// GetBestHeight returns the height of the last block
func (bc *Blockchain) GetBestHeight() int {
	var lastBlock Block
	if err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte(tipDbKey))
		blockData := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockData)
		return nil
	}); err != nil {
		log.Panic(err)
	}

	return lastBlock.Height
}

// GetBlock finds a block by its hash and return it
func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block Block
	if err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockData := b.Get(blockHash)
		if blockData == nil {
			return errors.New("block is not found")
		}

		block = *DeserializeBlock(blockData)
		return nil

	}); err != nil {
		return block, err
	}

	return block, nil
}

// GetBlockHashes returns a list of hashes of all blocks in the chain
func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	bci := bc.Iterator()
	for {
		block := bci.Next()
		blocks = append(blocks, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

// MineBlock mines a new block with the provided transaction
func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	for _, tx := range transactions {
		// TODO: ignore transaction which is not valid
		if !bc.VerifyTransaction(tx) {
			log.Panic("ERROR: Invalid transaction")
		}
	}

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte(tipDbKey))

		blockData := b.Get(lastHash)
		block := DeserializeBlock(blockData)
		lastHeight = block.Height

		return nil
	})

	newBlock := NewBlock(transactions, lastHash, lastHeight+1)

	if err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if err = b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
			log.Panic(err)
		}

		if err = b.Put([]byte(tipDbKey), newBlock.Hash); err != nil {
			log.Panic(err)
		}

		bc.tip = newBlock.Hash
		return nil
	}); err != nil {
		log.Panic(err)
	}

	return newBlock
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

// FindUTXO finds all unspent transactions
func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	utxo := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		b := bci.Next()

		for _, tx := range b.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.VOut {
				// check the output is spent?
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := utxo[txID]
				outs.Outputs = append(outs.Outputs, out)
				utxo[txID] = outs
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.VIn {
					inTxID := hex.EncodeToString(in.TxID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.VOut)
				}
			}
		}

		if len(b.PrevBlockHash) == 0 {
			break
		}
	}

	return utxo
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

// VerifyTransaction verifies transaction input signatures
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.VIn {
		prevTX, err := bc.FindTransaction(vin.TxID)
		if err != nil {
			log.Panic(err)
		}

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

// dbExists returns whether database file is exists
func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}
