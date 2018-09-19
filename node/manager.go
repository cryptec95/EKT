package node

import (
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
)

const (
	NODE_ENV_FULL_SYNC = "full"
	NODE_ENV_DELEGETE  = "delegate"
)

var fullNode Node
var nodeEnv string

func Init(env string) {
	nodeEnv = env
	switch env {
	case NODE_ENV_FULL_SYNC:
		fullNode = NewFullMode(conf.EKTConfig)
	case NODE_ENV_DELEGETE:
		fullNode = NewDelegateNode(conf.EKTConfig)
	}
	go fullNode.StartNode()
}

func GetMainChain() *blockchain.BlockChain {
	return fullNode.GetBlockChain()
}

func SuggestFee() int64 {
	return 0
}

/*
	for delegate node
*/
func BlockFromPeer(block blockchain.Block) {
	fullNode.BlockFromPeer(block)
}

func VoteFromPeer(vote blockchain.BlockVote) {
	fullNode.VoteFromPeer(vote)
}

func VoteResultFromPeer(votes blockchain.Votes) {
	fullNode.VoteResultFromPeer(votes)
}

/*
	for all node
*/

func GetVoteResults(chainId int64, hash string) blockchain.Votes {
	return fullNode.GetVoteResults(chainId, hash)
}

func GetBlockByHeight(chainId, height int64) *blockchain.Header {
	return fullNode.GetHeaderByHeight(chainId, height)
}
