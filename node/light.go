package node

import (
	"fmt"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/consensus"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/encapdb"
	"github.com/EducationEKT/EKT/param"
	"time"
)

type FullNode struct {
	config     conf.EKTConf
	blockchain *blockchain.BlockChain
	dbft       *consensus.DbftConsensus
	client     ektclient.IClient
}

func NewFullMode(config conf.EKTConf) *FullNode {
	node := &FullNode{
		config:     config,
		blockchain: blockchain.NewBlockChain(),
		client:     ektclient.NewClient(param.MainChainDelegateNode),
	}
	node.dbft = consensus.NewDbftConsensus(node.blockchain)
	return node
}

func (node FullNode) StartNode() {
	fmt.Println("Start full sync node")
	node.recoverFromDB()
	node.loop()
}

func (node FullNode) GetBlockChain() *blockchain.BlockChain {
	return node.blockchain
}

func (node FullNode) recoverFromDB() {
	node.dbft.RecoverFromDB()
}

func (node FullNode) BlockFromPeer(block blockchain.Header) {
	return
}

func (node FullNode) VoteFromPeer(vote blockchain.BlockVote) {
	return
}

func (node FullNode) VoteResultFromPeer(votes blockchain.Votes) {
	return
}

func (node FullNode) GetVoteResults(chainId int64, hash string) blockchain.Votes {
	return encapdb.GetVoteResults(chainId, hash)
}

func (node FullNode) GetBlockByHeight(chainId, height int64) *blockchain.Block {
	return encapdb.GetBlockByHeight(chainId, height)
}

func (node FullNode) GetHeaderByHeight(chainId, height int64) *blockchain.Header {
	return encapdb.GetHeaderByHeight(chainId, height)
}

func (node FullNode) loop() {
	fail, failTime := false, 0

	for height := node.blockchain.GetLastHeight() + 1; ; {
		if fail {
			if failTime >= 3 {
				time.Sleep(blockchain.BackboneBlockInterval)
			} else {
				time.Sleep(blockchain.BackboneBlockInterval / 5)
			}
		}

		if node.dbft.SyncHeight(height) {
			fmt.Println("Synchronized block at", height, ".")
			height++
			fail, failTime = false, 0
		} else {
			fmt.Println("Get block failed")
			fail, failTime = true, failTime+1
		}
	}
}
