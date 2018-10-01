package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/EducationEKT/EKT/contract"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
)

const EMPTY_TX = "ca4510738395af1429224dd785675309c344b2b549632e20275c69b15ed1d210"

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
	if hex.EncodeToString(block.GetHeader().TxHash) == EMPTY_TX {
		return []userevent.Transaction{}
	} else if len(block.Transactions) == 0 {
		body, err := block.Miner.GetDBValue(hex.EncodeToString(block.GetHeader().TxHash))
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
	if hex.EncodeToString(block.GetHeader().TxHash) == EMPTY_TX {
		return []userevent.TransactionReceipt{}
	} else if len(block.TransactionReceipts) == 0 {
		body, err := block.Miner.GetDBValue(hex.EncodeToString(block.GetHeader().ReceiptHash))
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

func (block *Block) NewTransaction(tx userevent.Transaction) *userevent.TransactionReceipt {
	if len(tx.From) != 32 {
		return nil
	}
	switch len(tx.To) {
	case types.AddressLength:
		// TODO 1  !importent add nonce  user.nonce++
		// TODO 2 !importent gas->KET
		txs := make(userevent.SubTransactions, 0)
		subTx := userevent.NewSubTransaction(tx.TxId(), tx.From, tx.To, tx.Amount, tx.Data, tx.TokenAddress)
		txs = append(txs, *subTx)
		receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
		receipt.SubTransactions = txs
		receipt.Success = block.GetHeader().NewSubTransaction(txs)
		if !receipt.Success {
			receipt.FailType = userevent.FailType_NO_ENOUGH_AMOUNT
		}
		block.Transactions = append(block.Transactions, tx)
		block.TransactionReceipts = append(block.TransactionReceipts, receipt)
		return &receipt
	case types.ContractAddressLength:
		block.GetHeader().CheckContractTransfer(tx)
		to, err := block.GetHeader().GetAccount(tx.To[:32])
		if err == nil && to != nil {
			if _, exist := to.Contracts[hex.EncodeToString(tx.To[32:])]; !exist {
				contract.InitContract(tx, to)
			}
		} else {
			to = types.NewAccount(tx.To[:32])
			contract.InitContract(tx, to)
		}
		if _, exist := to.Contracts[hex.EncodeToString(tx.To[32:])]; !exist {
			log.Crit("Invalid tx, %s", string(tx.Bytes()))
			return nil
		} else {
			receipt, data := contract.Run(tx, to)
			ca := to.Contracts[hex.EncodeToString(tx.To[32:])]
			ca.ContractData = data
			to.Contracts[hex.EncodeToString(tx.To[32:])] = ca
			block.GetHeader().StatTree.MustInsert(to.Address, to.ToBytes())
			if !block.CheckSubTransaction(tx, receipt.SubTransactions) {
				receipt = userevent.ContractRefuseTx(tx)
			}
			txs := make(userevent.SubTransactions, 0)
			subTx := userevent.NewSubTransaction(tx.TxId(), tx.From, tx.To, tx.Amount, tx.Data, tx.TokenAddress)
			txs = append(txs, *subTx)
			txs = append(txs, receipt.SubTransactions...)
			receipt.SubTransactions = txs
			block.GetHeader().NewSubTransaction(txs)
			block.Transactions = append(block.Transactions, tx)
			block.TransactionReceipts = append(block.TransactionReceipts, *receipt)
			return receipt
		}
	}
	return nil
}

func (block *Block) CheckSubTransaction(tx userevent.Transaction, subTxs userevent.SubTransactions) bool {
	if len(subTxs) > 0 {
		for _, subTx := range subTxs {
			if !bytes.EqualFold(subTx.From, tx.To) {
				return false
			}
		}
	}
	return true
}

func (block *Block) Finish() {
	block.header.UpdateMiner()
	block.header.TxHash = crypto.Sha3_256(block.Transactions.Bytes())
	db.GetDBInst().Set(block.header.TxHash, block.Transactions.Bytes())
	block.header.ReceiptHash = crypto.Sha3_256(block.TransactionReceipts.Bytes())
	db.GetDBInst().Set(block.header.ReceiptHash, block.TransactionReceipts.Bytes())
	block.Hash = block.header.CaculateHash()
}

func (block Block) ValidateNextBlock(next Block) bool {
	newBlock := CreateBlock(*(block.GetHeader()), next.Miner)
	for _, tx := range next.GetTransactions() {
		newBlock.NewTransaction(tx)
	}
	newBlock.Finish()
	return bytes.EqualFold(newBlock.Hash, next.Hash)
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
