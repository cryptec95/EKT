package ektclient

import (
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
)

var client IClient

func GetInst() IClient {
	return client
}

type IClient interface {
	// block
	GetHeaderByHeight(height int64) *blockchain.Header
	GetBlockByHeight(height int64) *blockchain.Block
	GetLastBlock() *blockchain.Header
	GetHeaderByHash(hash []byte) *blockchain.Header

	// vote
	GetVotesByBlockHash(hash string) blockchain.Votes

	// delegate
	BroadcastBlock(block blockchain.Block)
	SendVote(vote blockchain.PeerBlockVote)
	SendVoteResult(votes blockchain.Votes)

	// transaction
	GetSuggestionFee() int64
	SendTransaction(tx userevent.Transaction) error
	GetReceipt(txHash string) *userevent.ReceiptDetail

	// account
	GetAccountNonce(address string) int64
	GetGenesisAccounts() []types.Account

	GetValueByHash(hash []byte) []byte
}
