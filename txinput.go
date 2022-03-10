package blockchain

// TXInput represents a transaction input
type TXInput struct {
	TxID      []byte
	VOut      int
	Signature []byte
	PubKey    []byte
}
