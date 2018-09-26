package node

import (
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
)

type Node interface {
	StartNode()

	GetBlockChain() *blockchain.BlockChain
	GetVoteResults(chainId int64, hash string) blockchain.Votes
	GetHeaderByHeight(chainId, height int64) *blockchain.Header
	GetBlockByHeight(chainId, height int64) *blockchain.Block

	BlockFromPeer(block blockchain.Block)
	VoteFromPeer(vote blockchain.PeerBlockVote)
	VoteResultFromPeer(votes blockchain.Votes)
	Heartbeat(heartbeat types.Heartbeat)
}
