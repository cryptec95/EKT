package mode

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

type FullSyncNode struct {
	config     conf.EKTConf
	blockchain *blockchain.BlockChain
	consensus  i_consensus.Consensus
	client     ektclient.IClient
}

func NewFullSyncMode(config conf.EKTConf) *FullSyncNode {
	return &FullSyncNode{
		config:     config,
		blockchain: blockchain.NewBlockChain(),
		consensus:  consensus.NewBasicConsensus(),
		client:     ektclient.NewClient(param.MainChainDelegateNode),
	}
}

func (fullNode FullSyncNode) StartNode() {
	blockchain_manager.MainBlockChain = fullNode.blockchain
	fmt.Println("Start full sync node")
	fullNode.recoverFromDB()
	fullNode.loop()
}

func (fullNode FullSyncNode) recoverFromDB() {
	block, err := fullNode.blockchain.LastBlockFromDB()
	if err != nil || block == nil {
		// 将创世块写入数据库
		accounts := fullNode.config.GenesisBlockAccounts
		block = blockchain.GenesisBlock(accounts)
		fullNode.blockchain.SaveBlock(*block)
	} else {
		fullNode.blockchain.SetLastBlock(*block)
	}
	log.Info("Recovered from local database.")
}

func (fullNode FullSyncNode) loop() {
	fail, failTime := false, 0

	for height := fullNode.blockchain.GetLastHeight() + 1; ; {
		if fail {
			if failTime >= 3 {
				time.Sleep(blockchain.BackboneBlockInterval)
			} else {
				time.Sleep(blockchain.BackboneBlockInterval / 5)
			}
		}

		if block := fullNode.client.GetBlockByHeight(height); block != nil {
			hash := block.CurrentHash
			if !bytes.EqualFold(hash, block.CaculateHash()) {
				fail, failTime = true, failTime+1
				continue
			}

			votes := fullNode.client.GetVotesByBlockHash(hex.EncodeToString(block.CaculateHash()))
			if votes.Validate() {
				lastBlock := fullNode.blockchain.LastBlock()
				eventIds := fullNode.client.GetEventIds(block.Body)
				if len(eventIds) == 0 && !bytes.EqualFold(block.Body, hexutil.MustDecode("0xa3284248847e7ff1ac740635e292f219ea7f943a22ebdd733d2af173820bc291")) {
					continue
				}
				events, err := fullNode.client.GetEvents(eventIds)
				if err != nil {
					fail, failTime = true, failTime+1
					continue
				}
				if lastBlock.ValidateNextBlock(*block, events) {
					fmt.Println("Synchronized block at", block.Height, ".")
					fail, failTime = false, 0
					height++
					SaveVotes(votes)
					fullNode.blockchain.SaveBlock(*block)
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
