package node

import (
	"encoding/hex"
	"strings"

	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/param"
)

const (
	NODE_ENV_FULL_SYNC = "full"
	NODE_ENV_DELEGETE  = "delegate"
	Adaptive           = "adaptive"
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
	case Adaptive:
		env := checkEnv()
		Init(env)
	}
	fullNode.StartNode()
}

func checkEnv() string {
	for _, peer := range param.MainChainDelegateNode {
		if peer.Equal(conf.EKTConfig.Node) {
			pub, err := crypto.PubKey(conf.EKTConfig.PrivateKey)
			if err != nil {
				return NODE_ENV_FULL_SYNC
			}
			addr := types.FromPubKeyToAddress(pub)
			if strings.EqualFold(conf.EKTConfig.Node.Account, hex.EncodeToString(addr)) {
				return NODE_ENV_DELEGETE
			}
		}
	}
	return NODE_ENV_FULL_SYNC
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
func BlockFromPeer(clog *ctxlog.ContextLog, block *blockchain.Block) {
	fullNode.BlockFromPeer(clog, block)
}

func VoteFromPeer(vote blockchain.PeerBlockVote) {
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
