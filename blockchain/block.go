package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/EducationEKT/EKT/downloader"
	"strconv"
	"time"

	"github.com/EducationEKT/EKT/context"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/vm"
)

const (
	VM_CALL_TIMEOUT = 200 * time.Millisecond
	EMPTY_TX        = "ca4510738395af1429224dd785675309c344b2b549632e20275c69b15ed1d210"
)

type IBlock interface {
	GetTransactions() []userevent.Transaction
	GetTxReceipts() []userevent.TransactionReceipt
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
		body := downloader.Synchronise(block.GetHeader().TxHash)
		if !bytes.Equal(crypto.Sha3_256(body), block.GetHeader().TxHash) {
			return nil
		}
		var txs []userevent.Transaction
		err := json.Unmarshal(body, &txs)
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
		body := downloader.Synchronise(block.GetHeader().ReceiptHash)
		if !bytes.Equal(crypto.Sha3_256(body), block.GetHeader().ReceiptHash) {
			return nil
		}
		var receipts []userevent.TransactionReceipt
		err := json.Unmarshal(body, &receipts)
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
		return block.DeployContract(tx)
	case 65:
		// upgrade
		return block.UpgradeContract(tx)
	case types.AccountAddressLength:
		return block.NormalTransfer(tx)
	case types.ContractAddressLength:
		return block.ContractCall(tx)
	default:
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INVALID_ADDRESS)
		return &receipt
	}
}

func (block *Block) ContractCall(tx userevent.Transaction) *userevent.TransactionReceipt {
	toAccountAddress, toContractAddress := tx.To[:32], tx.To[32:]
	to, err := block.GetHeader().GetAccount(toAccountAddress)
	if err != nil || to == nil {
		to = types.NewAccount(toAccountAddress)
	}
	if _, exist := to.Contracts[hex.EncodeToString(toContractAddress)]; !exist {
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INVALID_ADDRESS)
		return &receipt
	}
	_vm := vm.NewVM(block.GetHeader(), db.GetDBInst())
	contractAccount := to.Contracts[hex.EncodeToString(toContractAddress)]
	txs, data, err := _vm.ContractCall(tx, VM_CALL_TIMEOUT)
	if err != nil {
		if err == vm.TIMEOUT_ERROR {
			receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_TIMEOUT)
			return &receipt
		}
		return userevent.ContractRefuseTx(tx)
	}
	contractAccount.ContractData.Contract = string(data)
	to.Contracts[hex.EncodeToString(toContractAddress)] = contractAccount
	block.GetHeader().StatTree.MustInsert(toAccountAddress, to.ToBytes())

	if !block.CheckSubTransaction(tx, txs) {
		_receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CHECK_CONTRACT_SUBTX_ERROR)
		return &_receipt
	}
	subTx := userevent.NewSubTransaction(tx.TxId(), tx.From, tx.To, tx.Amount, tx.Data, tx.TokenAddress)
	txs = append(txs, *subTx)
	receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
	receipt.Success = block.GetHeader().NewSubTransaction(txs)
	receipt.SubTransactions = txs
	return &receipt
}

func (block *Block) NormalTransfer(tx userevent.Transaction) *userevent.TransactionReceipt {
	txs := make(userevent.SubTransactions, 0)
	subTx := userevent.NewSubTransaction(tx.TxId(), tx.From, tx.To, tx.Amount, tx.Data, tx.TokenAddress)
	txs = append(txs, *subTx)
	receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
	receipt.SubTransactions = txs
	block.GetHeader().NewSubTransaction(txs)
	return &receipt
}

func (block *Block) DeployContract(tx userevent.Transaction) *userevent.TransactionReceipt {
	account, _ := block.GetHeader().GetAccount(tx.From)

	_vm := vm.NewVM(block.GetHeader(), db.GetDBInst())
	contractData, err := _vm.InitContractWithTimeout([]byte(tx.Data), VM_CALL_TIMEOUT)
	if err != nil {
		if err == vm.TIMEOUT_ERROR {
			receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_TIMEOUT)
			return &receipt
		} else {
			receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INIT_CONTRACT_ACCOUNT_FAIL)
			return &receipt
		}
	}

	contractHash := crypto.Sha3_256([]byte(tx.Data))
	db.GetDBInst().Set(contractHash, []byte(tx.Data))
	addr := crypto.Sha3_256([]byte(strconv.Itoa(len(account.Contracts) + 1)))

	contractAccount := types.NewContractAccount(addr, contractHash, *contractData)
	if account.Contracts == nil {
		account.Contracts = make(map[string]types.ContractAccount)
	}

	if tx.Amount > 0 && tx.TokenAddress == "" && account.GetAmount() >= tx.Amount {
		account.Amount -= tx.Amount
		contractAccount.Amount += tx.Amount
	}

	account.Contracts[hex.EncodeToString(addr)] = *contractAccount
	logErr(block.GetHeader().StatTree.MustInsert(account.Address, account.ToBytes()))

	receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
	return &receipt
}

func (block *Block) UpgradeContract(tx userevent.Transaction) *userevent.TransactionReceipt {
	code := int(tx.To[64])
	switch code {
	case 1:
		return block.upgradeContract(tx)
	}
	receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INVALID_ADDRESS)
	return &receipt
}

func (block *Block) upgradeContract(tx userevent.Transaction) *userevent.TransactionReceipt {
	// validate author
	if !bytes.HasPrefix(tx.To, tx.From) {
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_INVALID_CONTRACT_ADDRESS)
		return &receipt
	}

	// validate contract account
	account, err := block.GetHeader().GetAccount(tx.From)
	if err != nil || account.Contracts == nil {
		return nil
	}

	contractAccount, exist := account.Contracts[hex.EncodeToString(tx.To[32:64])]
	if !exist {
		return nil
	}

	// check upgradable
	if !contractAccount.ContractData.Prop.Upgradable {
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_UPGRADE_REFUSED)
		return &receipt
	}

	_vm := vm.NewVM(block.GetHeader(), db.GetDBInst())
	contractData, err := _vm.UpgradeContract([]byte(tx.Data), &contractAccount.ContractData, VM_CALL_TIMEOUT)
	if err != nil {
		receipt := userevent.NewTransactionReceipt(tx, false, userevent.FailType_CONTRACT_ERROR)
		return &receipt
	}

	logErr(db.GetDBInst().Set(contractAccount.CodeHash, []byte(tx.Data)))

	contractAccount.CodeHash = crypto.Sha3_256([]byte(tx.Data))
	contractAccount.ContractData = *contractData

	account.Contracts[hex.EncodeToString(tx.To[32:64])] = contractAccount

	logErr(block.GetHeader().StatTree.MustInsert(tx.From, account.ToBytes()))

	receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
	return &receipt
}

func (block *Block) CheckSubTransaction(tx userevent.Transaction, subTxs userevent.SubTransactions) bool {
	if len(subTxs) > 0 {
		for _, subTx := range subTxs {
			if !bytes.Equal(subTx.From, tx.To) || subTx.Amount <= 0 {
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
	logErr(db.GetDBInst().Set(block.Header.TxHash, block.Transactions.Bytes()))
	block.Header.ReceiptHash = crypto.Sha3_256(block.TransactionReceipts.Bytes())
	logErr(db.GetDBInst().Set(block.Header.ReceiptHash, block.TransactionReceipts.Bytes()))
	block.Hash = block.Header.CalculateHash()
	logErr(db.GetDBInst().Set(block.Hash, block.Header.Bytes()))
}

func logErr(err error) {
	if err != nil {
		log.Debug("error: %v", err)
	}
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
	header := NewHeader_V2(last, time, last.CalculateHash(), coinbase)
	return &Block{
		Header:              header,
		Miner:               peer,
		Transactions:        make([]userevent.Transaction, 0),
		TransactionReceipts: make([]userevent.TransactionReceipt, 0),
	}
}

func NewBlock_V2(last Header, time int64, peer types.Peer) *Block {
	coinbase, _ := hex.DecodeString(peer.Account)
	header := NewHeader_V2(last, time, last.CalculateHash(), coinbase)
	return &Block{
		Header:              header,
		Miner:               peer,
		Transactions:        make([]userevent.Transaction, 0),
		TransactionReceipts: make([]userevent.TransactionReceipt, 0),
	}
}
