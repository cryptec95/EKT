package node

import (
	"time"

	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/consensus"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/encapdb"
)

type ForkNode struct {
	blockchain *blockchain.BlockChain
	dbft       *consensus.DbftConsensus
	client     ektclient.IClient
}

func NewForkNode() *ForkNode {
	node := &ForkNode{
		blockchain: blockchain.NewBlockChain(1),
		client:     ektclient.GetInst(),
	}
	node.dbft = consensus.NewDbftConsensus(node.blockchain, node.client)
	return node
}

func (node ForkNode) StartNode() {
	node.recoverFromDB()
	go node.loop()
}

func (node ForkNode) GetBlockChain() *blockchain.BlockChain {
	return node.blockchain
}

func (node ForkNode) recoverFromDB() {
	node.dbft.RecoverFromDB()
}

func (node ForkNode) BlockFromPeer(clog *ctxlog.ContextLog, block *blockchain.Block) {
	return
}

func (node ForkNode) VoteFromPeer(vote blockchain.PeerBlockVote) {
	return
}

func (node ForkNode) VoteResultFromPeer(votes blockchain.Votes) {
	return
}

func (node ForkNode) GetVoteResults(chainId int64, hash string) blockchain.Votes {
	return encapdb.GetVoteResults(chainId, hash)
}

func (node ForkNode) GetBlockByHeight(chainId, height int64) *blockchain.Block {
	return encapdb.GetBlockByHeight(chainId, height)
}

func (node ForkNode) GetHeaderByHeight(chainId, height int64) *blockchain.Header {
	return encapdb.GetHeaderByHeight(chainId, height)
}

func (node ForkNode) loop() {
	fail, failTime := false, 0

	for height := node.blockchain.GetLastHeight(); ; {
		if fail {
			if failTime >= 3 {
				time.Sleep(blockchain.BackboneBlockInterval)
			}
		}

		if node.dbft.ForkSync(height) {
			height++
			fail, failTime = false, 0
		} else {
			fail, failTime = true, failTime+1
		}
	}
}
