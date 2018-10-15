package node

import (
	"encoding/hex"
	"time"

	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/consensus"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/encapdb"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/param"
)

type DelegateNode struct {
	db         db.IKVDatabase
	config     conf.EKTConf
	blockchain *blockchain.BlockChain
	dbft       *consensus.DbftConsensus
	seated     bool
	client     ektclient.IClient
}

func NewDelegateNode(conf conf.EKTConf) *DelegateNode {
	node := &DelegateNode{
		db:         db.GetDBInst(),
		config:     conf,
		seated:     false,
		blockchain: blockchain.NewBlockChain(1),
		client:     ektclient.NewClient(param.MainChainDelegateNode),
	}
	node.dbft = consensus.NewDbftConsensus(node.blockchain, node.client)
	return node
}

func (delegate DelegateNode) StartNode() {
	delegate.RecoverFromDB()
	go delegate.sync()
}

func (delegate DelegateNode) sync() {
	lastHeight := delegate.dbft.Blockchain.GetLastHeight()
	fail, failTime := false, 0
	for {
		if fail {
			time.Sleep(time.Second)
		}
		height := delegate.dbft.Blockchain.GetLastHeight()
		if height == lastHeight {
			if fail && failTime >= 3 {
				if !delegate.seated {
					go delegate.tryPack()
				}
			}
			log.Debug("Height has not change for an interval, synchronizing block.")
			if delegate.dbft.SyncHeight(lastHeight + 1) {
				log.Debug("Synchronized block at lastHeight %d.", lastHeight+1)
				fail, failTime = false, 0
				lastHeight = delegate.dbft.Blockchain.GetLastHeight()
			} else {
				fail, failTime = true, failTime+1
				log.Debug("Synchronize block at lastHeight %d failed.", lastHeight+1)
				time.Sleep(blockchain.BackboneBlockInterval)
			}
		} else {
			lastHeight = height
		}
	}
}

func (delegate DelegateNode) tryPack() {
	if !delegate.seated {
		if delegate.dbft.TryPack() {
			delegate.seated = true
		}
	}
}

func (delegate DelegateNode) GetBlockChain() *blockchain.BlockChain {
	return delegate.blockchain
}

func (delegate DelegateNode) RecoverFromDB() {
	delegate.dbft.RecoverFromDB()
}

func (delegate DelegateNode) BlockFromPeer(clog *ctxlog.ContextLog, block *blockchain.Block) {
	clog.Log("blockHash", hex.EncodeToString(block.Hash))
	if block.GetHeader().Height != delegate.blockchain.LastHeader().Height+1 {
		clog.Log("Invalid height", true)
		return
	}
	delegate.dbft.BlockFromPeer(clog, block)
}

func (delegate DelegateNode) VoteFromPeer(vote blockchain.PeerBlockVote) {
	delegate.dbft.VoteFromPeer(vote)
}

func (delegate DelegateNode) VoteResultFromPeer(votes blockchain.Votes) {
	delegate.dbft.ReceiveVoteResult(votes)
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
