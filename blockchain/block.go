package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
)

type IBlock interface {
	GetHeader() Header
	GetTransactions() []userevent.Transaction
	GetTxReceipts() []userevent.TransactionReceipt
	ValidateHash() bool
}

type Block struct {
	Header              *Header                `json:"-"`
	Hash                types.HexBytes         `json:"hash"`
	Signature           types.HexBytes         `json:"signature"`
	Miner               types.Peer             `json:"miner"`
	Transactions        userevent.Transactions `json:"-"`
	TransactionReceipts userevent.Receipts     `json:"-"`
}

func GetBlockFromBytes(data []byte) *Block {
	var block Block
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil
	}
	return &block
}

func (block Block) GetTransactions() []userevent.Transaction {
	body, err := block.Miner.GetDBValue(hex.EncodeToString(block.Header.TxHash))
	if err != nil {
		return nil
	}
	var txs []userevent.Transaction
	err = json.Unmarshal(body, &txs)
	if err != nil {
		return nil
	}
	return txs
}

func (block Block) GetTxReceipts() []userevent.TransactionReceipt {
	body, err := block.Miner.GetDBValue(hex.EncodeToString(block.Header.ReceiptHash))
	if err != nil {
		return nil
	}
	var receipts []userevent.TransactionReceipt
	err = json.Unmarshal(body, &receipts)
	if err != nil {
		return nil
	}
	return receipts
}

func (block Block) GetHeader() *Header {
	if block.Header != nil {
		return block.Header
	} else {
		data, err := block.Miner.GetDBValue(hex.EncodeToString(block.Hash))
		if err != nil {
			return nil
		}
		header := FromBytes2Header(data)
		return header
	}
}

func (block *Block) NewTransaction(tx userevent.Transaction) {
	if len(tx.From) != 32 || len(tx.To) != 32 {
		return
	}

	receipt := block.Header.NewTransaction(tx)
	if receipt.Success {
		block.Header.TotalFee += tx.Fee
	}
	block.Transactions = append(block.Transactions, tx)
	block.TransactionReceipts = append(block.TransactionReceipts, receipt)
}

func (block *Block) Finish() {
	block.Header.UpdateMiner()
	block.Header.TxHash = crypto.Sha3_256(block.Transactions.Bytes())
	block.Header.ReceiptHash = crypto.Sha3_256(block.TransactionReceipts.Bytes())
	block.Hash = block.Header.CaculateHash()
}

func (block *Block) Sign(privKey []byte) {
	block.Signature, _ = crypto.Crypto(block.Hash, privKey)
}

func CreateGenesisBlock(accounts []types.Account) Block {
	header := GenesisHeader(accounts)
	block := Block{
		Header: header,
	}
	return block
}

func CreateBlock(last Block, peer types.Peer) Block {
	coinbase, _ := hex.DecodeString(peer.Account)
	header := NewHeader(*last.Header, last.Hash, coinbase)
	return Block{
		Header:              header,
		Miner:               peer,
		Transactions:        make([]userevent.Transaction, 0),
		TransactionReceipts: make([]userevent.TransactionReceipt, 0),
	}
}
