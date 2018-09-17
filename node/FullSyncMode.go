package node

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/blockchain_manager"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/consensus"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/i_consensus"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/param"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"time"
)

type FullNode struct {
	config     conf.EKTConf
	blockchain *blockchain.BlockChain
	consensus  i_consensus.Consensus
	client     ektclient.IClient
}

func NewFullMode(config conf.EKTConf) *FullNode {
	return &FullNode{
		config:     config,
		blockchain: blockchain.NewBlockChain(),
		consensus:  consensus.NewBasicConsensus(),
		client:     ektclient.NewClient(param.MainChainDelegateNode),
	}
}

func (node FullNode) StartNode() {
	blockchain_manager.MainBlockChain = node.blockchain
	fmt.Println("Start full sync node")
	node.recoverFromDB()
	node.loop()
}

func (node FullNode) recoverFromDB() {
	block, err := node.blockchain.LastBlockFromDB()
	if err != nil || block == nil {
		// 将创世块写入数据库
		accounts := node.config.GenesisBlockAccounts
		block = blockchain.GenesisBlock(accounts)
		node.blockchain.SaveBlock(*block)
	} else {
		node.blockchain.SetLastBlock(*block)
	}
	log.Info("Recovered from local database.")
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

		if block := node.client.GetBlockByHeight(height); block != nil {
			hash := block.CurrentHash
			if !bytes.EqualFold(hash, block.CaculateHash()) {
				fail, failTime = true, failTime+1
				continue
			}

			votes := node.client.GetVotesByBlockHash(hex.EncodeToString(block.CaculateHash()))
			if votes.Validate() {
				lastBlock := node.blockchain.LastBlock()
				eventIds := node.client.GetEventIds(block.Body)
				if len(eventIds) == 0 && !bytes.EqualFold(block.Body, hexutil.MustDecode("0xa3284248847e7ff1ac740635e292f219ea7f943a22ebdd733d2af173820bc291")) {
					continue
				}
				events, err := node.client.GetEvents(eventIds)
				if err != nil {
					fail, failTime = true, failTime+1
					continue
				}
				if lastBlock.ValidateNextBlock(*block, events) {
					fmt.Println("Synchronized block at", block.Height, ".")
					fail, failTime = false, 0
					height++
					SaveVotes(votes)
					node.blockchain.SaveBlock(*block)
				} else {
					fail, failTime = true, failTime+1
				}
			} else {
				fail, failTime = true, failTime+1
			}
		} else {
			fail, failTime = true, failTime+1
		}
	}
}
