package blockchain

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header and/or uncle verification.
type ChainReader interface {
	// Get last block
	LastBlock() Block

	// Get block by height
	GetBlockByHeight(height int64) *Block

	// get block by block hash
	GetBlockByHash(hash []byte) *Block
}

type ChainWriter interface {
	// add this block to the end of chain, return error if block height is not right
	SetLastBlock(block Block) error

	// insert the block to specific position and abort exist block which height is more than this block
	SetBlock(block Block) (numAborted int, err error)
}
