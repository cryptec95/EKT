package i_consensus

import "github.com/EducationEKT/EKT/blockchain"

type Consensus interface {
	// 验证打包时间、签名等
	VerifyAuthor(block blockchain.Block, blockchain blockchain.BlockChain)

	// 验证默克尔树是否正确
	VerifyState(block blockchain.Block, blockchain blockchain.BlockChain)

	// 验证是否可以写入区跨链中
	VerifyHeader(block blockchain.Block, blockchain blockchain.BlockChain)
}
