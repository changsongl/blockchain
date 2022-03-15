package blockchain

// TXOutput represents a transaction outpu
type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	out.PubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
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
