package consensus

import (
	"bytes"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
)

type BasicConsensus struct{}

func NewBasicConsensus() BasicConsensus {
	return BasicConsensus{}
}

func (consensus BasicConsensus) verifyAuthor(block blockchain.Block, signature []byte) bool {
	pubKey, err := crypto.RecoverPubKey(crypto.Sha3_256(block.CurrentHash), signature)
	if err != nil {
		return false
	}
	if !bytes.EqualFold(types.FromPubKeyToAddress(pubKey), block.Coinbase) {
		return false
	}
	return true
}

// 验证默克尔树是否正确
func (consensus BasicConsensus) verifyNextBlock(block blockchain.Block, lastBlock blockchain.Block, events []userevent.IUserEvent) bool {
	return lastBlock.ValidateNextBlock(block, events)
}

// 验证是否可以写入区跨链中
func (consensus BasicConsensus) VerifyHeader(block blockchain.Block, reader blockchain.ChainReader, events []userevent.IUserEvent) bool {
	if !consensus.verifyAuthor(block, block.Signature) {
		return false
	}
	lastBlock := reader.GetBlockByHeight(block.Height - 1)
	if lastBlock == nil || !consensus.verifyNextBlock(block, *lastBlock, events) {
		return false
	}
	return true
}
