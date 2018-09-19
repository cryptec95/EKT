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
	header              *Header                `json:"-"`
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
	if len(block.Transactions) == 0 {
		body, err := block.Miner.GetDBValue(hex.EncodeToString(block.header.TxHash))
		if err != nil {
			return nil
		}
		var txs []userevent.Transaction
		err = json.Unmarshal(body, &txs)
		if err != nil {
			return nil
		}
		block.Transactions = txs
	}
	return block.Transactions
}

func (block Block) GetTxReceipts() []userevent.TransactionReceipt {
	if len(block.TransactionReceipts) == 0 {
		body, err := block.Miner.GetDBValue(hex.EncodeToString(block.header.ReceiptHash))
		if err != nil {
			return nil
		}
		var receipts []userevent.TransactionReceipt
		err = json.Unmarshal(body, &receipts)
		if err != nil {
			return nil
		}
		block.TransactionReceipts = receipts
	}
	return block.TransactionReceipts
}

func (block Block) GetHeader() *Header {
	if block.header != nil {
		return block.header
	} else {
		data, err := block.Miner.GetDBValue(hex.EncodeToString(block.Hash))
		if err != nil {
			return nil
		}
		block.header = FromBytes2Header(data)
	}
	return block.header
}

func (block *Block) NewTransaction(tx userevent.Transaction) {
	if len(tx.From) != 32 || len(tx.To) != 32 {
		return
	}
	receipt := block.header.NewTransaction(tx)
	if receipt.Success {
		block.header.TotalFee += tx.Fee
	}
	block.Transactions = append(block.Transactions, tx)
	block.TransactionReceipts = append(block.TransactionReceipts, receipt)
}

func (block *Block) Finish() {
	block.header.UpdateMiner()
	block.header.TxHash = crypto.Sha3_256(block.Transactions.Bytes())
	block.header.ReceiptHash = crypto.Sha3_256(block.TransactionReceipts.Bytes())
	block.Hash = block.header.CaculateHash()
}

func (block *Block) Sign(privKey []byte) error {
	sign, err := crypto.Crypto(block.Hash, privKey)
	block.Signature = sign
	return err
}

func (block Block) Bytes() []byte {
	data, _ := json.Marshal(block)
	return data
}

func CreateGenesisBlock(accounts []types.Account) Block {
	header := GenesisHeader(accounts)
	block := Block{
		header: header,
	}
	return block
}

func CreateBlock(last Header, peer types.Peer) *Block {
	coinbase, _ := hex.DecodeString(peer.Account)
	header := NewHeader(last, last.CaculateHash(), coinbase)
	return &Block{
		header:              header,
		Miner:               peer,
		Transactions:        make([]userevent.Transaction, 0),
		TransactionReceipts: make([]userevent.TransactionReceipt, 0),
	}
}
