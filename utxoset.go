package blockchain

// UTXOSet represents UTXO set
type UTXOSet struct {
	Blockchain *Blockchain
}

// FindSpendableOutputs finds and returns unspent outputs to reference in inputs TODO: implements
func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	return 0, nil
}
