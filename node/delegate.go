package node

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/consensus"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/encapdb"
	"github.com/EducationEKT/EKT/param"
)

type DelegateNode struct {
	db         db.IKVDatabase
	config     conf.EKTConf
	blockchain *blockchain.BlockChain
	dbft       *consensus.DbftConsensus
	client     ektclient.IClient
}

func NewDelegateNode(conf conf.EKTConf) *DelegateNode {
	node := &DelegateNode{
		db:         db.GetDBInst(),
		config:     conf,
		blockchain: blockchain.NewBlockChain(1),
		client:     ektclient.NewClient(param.MainChainDelegateNode),
	}
	node.dbft = consensus.NewDbftConsensus(node.blockchain, node.client)
	return node
}

func (delegate DelegateNode) StartNode() {
	delegate.RecoverFromDB()
	delegate.dbft.Run()
}

func (delegate DelegateNode) GetBlockChain() *blockchain.BlockChain {
	return delegate.blockchain
}

func (delegate DelegateNode) RecoverFromDB() {
	delegate.dbft.RecoverFromDB()
}

func (delegate DelegateNode) Heartbeat(heartbeat types.Heartbeat) {
	if heartbeat.Validate() {
		delegate.dbft.ReceiveHeartbeat(heartbeat)
	}
}

func (delegate DelegateNode) BlockFromPeer(block blockchain.Block) {
	ctxLog := ctxlog.NewContextLog("blockFromPeer")
	defer ctxLog.Finish()
	ctxLog.Log("blockHash", hex.EncodeToString(block.Hash))
	delegate.dbft.BlockFromPeer(ctxLog, block)
}

func (delegate DelegateNode) VoteFromPeer(vote blockchain.PeerBlockVote) {
	delegate.dbft.VoteFromPeer(vote)
}

func (delegate DelegateNode) VoteResultFromPeer(votes blockchain.Votes) {
	delegate.dbft.RecieveVoteResult(votes)
}

func (delegate DelegateNode) GetVoteResults(chainId int64, hash string) blockchain.Votes {
	return encapdb.GetVoteResults(chainId, hash)
}

func (delegate DelegateNode) GetBlockByHeight(chainId, height int64) *blockchain.Block {
	return encapdb.GetBlockByHeight(chainId, height)
}

func (delegate DelegateNode) GetHeaderByHeight(chainId, height int64) *blockchain.Header {
	return encapdb.GetHeaderByHeight(chainId, height)
}
