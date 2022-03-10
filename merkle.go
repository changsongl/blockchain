package blockchain

import "crypto/sha256"

// MerkleTree represent a Merkle tree
type MerkleTree struct {
	RootNode *MerkleNode
}

// MerkleNode represent a Merkle tree node
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// NewMerkleTree creates a new Merkle tree from a sequence of data
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*MerkleNode

	l := len(data)

	if l%2 != 0 {
		data = append(data, data[l-1])
		l++
	}

	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, node)
	}

	for i := 0; i < l/2; i++ {
		var newLevel []*MerkleNode

		for j := 0; i < len(nodes); j += 2 {
			node := NewMerkleNode(nodes[j], nodes[j+1], nil)
			newLevel = append(newLevel, node)
		}

		nodes = newLevel
	}

	if len(nodes) == 0 {
		return &MerkleTree{nil}
	}

	return &MerkleTree{nodes[0]}
}

// NewMerkleNode creates a new Merkle tree node
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := &MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else if left == nil || right == nil {
		panic("NewMerkleNode left or right is nil")
	} else {
		prevHash := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHash)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return mNode
}
