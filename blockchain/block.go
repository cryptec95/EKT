package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/EducationEKT/EKT/context"
	"github.com/EducationEKT/EKT/contract"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/util"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const EMPTY_TX = "ca4510738395af1429224dd785675309c344b2b549632e20275c69b15ed1d210"

type IBlock interface {
	GetHeader() Header
	GetTransactions() []userevent.Transaction
	GetTxReceipts() []userevent.TransactionReceipt
	ValidateHash() bool
}

type Block struct {
	Header              *Header                `json:"header"`
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
	return block.Header
}

func (block *Block) SetHeader(header *Header) {
	block.Header = header
}

func (block *Block) GetSticker() *context.Sticker {
	sticker := context.NewSticker()
	sticker.Save("lastHash", block.GetHeader().PreviousHash)
	sticker.Save("timestamp", block.GetHeader().Timestamp)
	sticker.Save("height", block.GetHeader().Height)
	return sticker
}

func (block *Block) NewTransaction(tx userevent.Transaction) *userevent.TransactionReceipt {
	if !block.GetHeader().CheckTransfer(tx) {
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CHECK_FAIL)
		return &receipt
	}
	switch len(tx.To) {
	case 0:
		// Deploy contract
		account, _ := block.GetHeader().GetAccount(tx.From)
		contractData, contractHash, err := contract.InitContract(block.GetSticker(), account, tx)
		if err != nil {
			receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_ERROR)
			return &receipt
		}
		addr, _ := hexutil.Decode(hexutil.EncodeUint64(uint64(len(account.Contracts) + 1)))
		addr = util.PendingLeft(addr, 32, byte(0))
		contractAccount := types.NewContractAccount(addr, contractHash, contractData)
		if account.Contracts == nil {
			account.Contracts = make(map[string]types.ContractAccount)
		}
		account.Contracts[hex.EncodeToString(addr)] = *contractAccount
		block.GetHeader().StatTree.MustInsert(account.Address, account.ToBytes())
		receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
		return &receipt
	case types.AccountAddressLength:
		txs := make(userevent.SubTransactions, 0)
		subTx := userevent.NewSubTransaction(tx.TxId(), tx.From, tx.To, tx.Amount, tx.Data, tx.TokenAddress)
		txs = append(txs, *subTx)
		receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
		receipt.SubTransactions = txs
		block.GetHeader().NewSubTransaction(txs)
		return &receipt
	case types.ContractAddressLength:
		toAccountAddress, toContractAddress := tx.To[:32], tx.To[32:]
		to, err := block.GetHeader().GetAccount(toAccountAddress)
		if err != nil || to == nil {
			to = types.NewAccount(toAccountAddress)
		}
		if _, exist := to.Contracts[hex.EncodeToString(toContractAddress)]; !exist {
			if !contract.InitContractAccount(tx, to) {
				receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INIT_CONTRACT_ACCOUNT_FAIL)
				return &receipt
			}
		}
		receipt, data := contract.Run(block.GetSticker(), tx, to)
		if !receipt.Success {
			_receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_ERROR)
			return &_receipt
		} else {
			contractAccount := to.Contracts[hex.EncodeToString(toContractAddress)]
			contractAccount.ContractData = data
			to.Contracts[hex.EncodeToString(toContractAddress)] = contractAccount
			block.GetHeader().StatTree.MustInsert(toAccountAddress, to.ToBytes())
		}
		if !block.CheckSubTransaction(tx, receipt.SubTransactions) {
			_receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_ERROR)
			return &_receipt
		}
		txs := make(userevent.SubTransactions, 0)
		subTx := userevent.NewSubTransaction(tx.TxId(), tx.From, tx.To, tx.Amount, tx.Data, tx.TokenAddress)
		txs = append(txs, *subTx)
		txs = append(txs, receipt.SubTransactions...)
		receipt.Success = block.GetHeader().NewSubTransaction(txs)
		receipt.SubTransactions = txs
		return receipt
	default:
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INVALID_ADDRESS)
		return &receipt
	}
}

func (block *Block) CheckSubTransaction(tx userevent.Transaction, subTxs userevent.SubTransactions) bool {
	if len(subTxs) > 0 {
		for _, subTx := range subTxs {
			if !bytes.EqualFold(subTx.From, tx.To) || subTx.Amount <= 0 {
				return false
			}
			subTx.Parent = tx.TxId()
		}
	}
	return true
}

func (block *Block) Finish() {
	block.Header.UpdateMiner()
	block.Header.TxHash = crypto.Sha3_256(block.Transactions.Bytes())
	db.GetDBInst().Set(block.Header.TxHash, block.Transactions.Bytes())
	block.Header.ReceiptHash = crypto.Sha3_256(block.TransactionReceipts.Bytes())
	db.GetDBInst().Set(block.Header.ReceiptHash, block.TransactionReceipts.Bytes())
	block.Hash = block.Header.CaculateHash()
	db.GetDBInst().Set(block.Hash, block.Header.Bytes())
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
		Header: header,
	}
	return block
}

func CreateBlock(last Header, time int64, peer types.Peer) *Block {
	coinbase, _ := hex.DecodeString(peer.Account)
	header := NewHeader(last, time, last.CaculateHash(), coinbase)
	return &Block{
		Header:              header,
		Miner:               peer,
		Transactions:        make([]userevent.Transaction, 0),
		TransactionReceipts: make([]userevent.TransactionReceipt, 0),
	}
}
