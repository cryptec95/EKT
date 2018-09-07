package mode

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/blockchain_manager"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/param"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"time"
)

type FullSyncMode struct {
	config     conf.EKTConf
	blockchain *blockchain.BlockChain
	client     ektclient.IClient
}

func NewFullSyncMode(config conf.EKTConf) *FullSyncMode {
	return &FullSyncMode{
		config:     config,
		blockchain: blockchain.NewBlockChain(),
		client:     ektclient.NewClient(param.MainChainDelegateNode),
	}
}

func (smode FullSyncMode) Start() {
	blockchain_manager.MainBlockChain = smode.blockchain
	fmt.Println("Start full sync mode")
	smode.recoverFromDB()
	smode.syncLoop()
}

func (smode FullSyncMode) recoverFromDB() {
	block, err := smode.blockchain.LastBlock()
	if err != nil || block == nil {
		// 将创世块写入数据库
		accounts := conf.EKTConfig.GenesisBlockAccounts
		block = blockchain.GenesisBlock(accounts)
		smode.blockchain.SaveBlock(*block)
	} else {
		smode.blockchain.SetLastBlock(*block)
	}
	log.Info("Recovered from local database.")
}

func (smode FullSyncMode) syncLoop() {
	fail, failTime := false, 0

	for height := smode.blockchain.GetLastHeight() + 1; ; {
		if fail {
			if failTime >= 3 {
				time.Sleep(blockchain.BackboneBlockInterval)
			} else {
				time.Sleep(blockchain.BackboneBlockInterval / 5)
			}
		}

		if block := smode.client.GetBlockByHeight(height); block != nil {
			hash := block.CurrentHash
			if !bytes.EqualFold(hash, block.CaculateHash()) {
				fail, failTime = true, failTime+1
				continue
			}

			votes := smode.client.GetVotesByBlockHash(hex.EncodeToString(block.CaculateHash()))
			if votes.Validate() {
				lastBlock := smode.blockchain.GetLastBlock()
				eventIds := smode.client.GetEventIds(block.Body)
				if len(eventIds) == 0 && !bytes.EqualFold(block.Body, hexutil.MustDecode("0xa3284248847e7ff1ac740635e292f219ea7f943a22ebdd733d2af173820bc291")) {
					continue
				}
				events, err := smode.client.GetEvents(eventIds)
				if err != nil {
					fail, failTime = true, failTime+1
					continue
				}
				if lastBlock.ValidateNextBlock(*block, events) {
					fmt.Println("Synchronized block at", block.Height, ".")
					fail, failTime = false, 0
					height++
					SaveVotes(votes)
					smode.blockchain.SaveBlock(*block)
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
