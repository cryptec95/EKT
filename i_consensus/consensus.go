package i_consensus

import (
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/userevent"
)

type Consensus interface {
	// 验证是否可以写入区跨链中
	VerifyHeader(block blockchain.Block, reader blockchain.ChainReader, events []userevent.IUserEvent) bool
}
