package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const subsidy = 10

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

// DeserializeTransaction deserializes a transaction
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&transaction); err != nil {
		log.Panic(err)
	}

	return transaction
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

// String returns a human-readable representation of a transaction
func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

	for i, input := range tx.VIn {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.TxID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.VOut))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.VOut {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
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

// Verify verifies signature of Transaction inputs
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	} else if err := tx.validatePrevTXs(prevTXs); err != nil {
		log.Panic(err)
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vIn := range tx.VIn {
		prevTX := prevTXs[hex.EncodeToString(vIn.TxID)]
		txCopy.VIn[inID].Signature = nil
		txCopy.VIn[inID].PubKey = prevTX.VOut[vIn.VOut].PubKeyHash

		r, s := &big.Int{}, &big.Int{}
		sigLen := len(vIn.Signature)
		r.SetBytes(vIn.Signature[:sigLen/2])
		s.SetBytes(vIn.Signature[sigLen/2:])

		x, y := &big.Int{}, &big.Int{}
		keyLen := len(vIn.PubKey)
		x.SetBytes(vIn.PubKey[:keyLen/2])
		y.SetBytes(vIn.PubKey[keyLen/2:])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		if ecdsa.Verify(rawPubKey, []byte(dataToVerify), r, s) == false {
			return false
		}

		txCopy.VIn[inID].PubKey = nil
	}

	return true
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}

		data = fmt.Sprintf("%x", randData)
	}

	txIn := TXInput{TxID: []byte{}, VOut: TransactionCoinbaseVInVOutDefault, Signature: nil, PubKey: []byte(data)}
	txOut := NewTXOutput(subsidy, to)
	tx := &Transaction{ID: nil, VIn: []TXInput{txIn}, VOut: []TXOutput{*txOut}}
	tx.ID = tx.Hash()

	return tx
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(wallet *Wallet, to string, amount int, utxoSet *UTXOSet) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	pubKeyHash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := utxoSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// builds a list of inputs
	for txID, outs := range validOutputs {
		txIDDecode, err := hex.DecodeString(txID)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TXInput{TxID: txIDDecode, VOut: out, Signature: nil, PubKey: wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	from := string(wallet.GetAddress())
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{ID: nil, VIn: inputs, VOut: outputs}
	tx.ID = tx.Hash()
	utxoSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return nil
}
