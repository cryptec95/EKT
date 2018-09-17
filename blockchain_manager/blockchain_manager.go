package blockchain_manager

import (
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/consensus"
)

var MainBlockChain *blockchain.BlockChain
var MainBlockChainConsensus *consensus.DbftConsensus

func Init() {
	MainBlockChain = blockchain.NewBlockChain()
	MainBlockChainConsensus = consensus.NewDbftConsensus(MainBlockChain)
	go MainBlockChainConsensus.StableRun()
}

func GetMainChain() *blockchain.BlockChain {
	return MainBlockChain
}

func GetMainChainConsensus() *consensus.DbftConsensus {
	return MainBlockChainConsensus
}
