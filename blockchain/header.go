package blockchain

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/EducationEKT/EKT/MPTPlus"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
)

const (
	HEADER_VERSION_PURE_MTP = 0
	HEADER_VERSION_MIXED    = 1
)

type Header struct {
	Height       int64          `json:"height"`
	Timestamp    int64          `json:"timestamp"`
	TotalFee     int64          `json:"totalFee"`
	PreviousHash types.HexBytes `json:"previousHash"`
	Coinbase     types.HexBytes `json:"miner"`
	StatTree     *MPTPlus.MTP   `json:"statRoot"`
	TokenTree    *MPTPlus.MTP   `json:"tokenRoot"`
	TxHash       types.HexBytes `json:"txHash"`
	ReceiptHash  types.HexBytes `json:"receiptHash"`
	Version      int            `json:"version"`
}

func (header *Header) Bytes() []byte {
	data, _ := json.Marshal(header)
	return data
}

func (header *Header) CaculateHash() []byte {
	return crypto.Sha3_256(header.Bytes())
}

func (header Header) GetAccount(address []byte) (*types.Account, error) {
	value, err := header.StatTree.GetValue(address)
	if err != nil {
		return nil, err
	}
	var account types.Account
	err = json.Unmarshal(value, &account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (header Header) ExistAddress(address []byte) bool {
	return header.StatTree.ContainsKey(address)
}

func GenesisHeader(accounts []types.Account) *Header {
	header := &Header{
		Height:       0,
		TotalFee:     0,
		PreviousHash: nil,
		Timestamp:    0,
		StatTree:     MPTPlus.NewMTP(db.GetDBInst()),
		TokenTree:    MPTPlus.NewMTP(db.GetDBInst()),
		Version:      HEADER_VERSION_MIXED,
	}

	for _, account := range accounts {
		header.StatTree.MustInsert(account.Address, account.ToBytes())
	}

	return header
}

func NewHeader(last Header, parentHash types.HexBytes, coinbase types.HexBytes) *Header {
	block := &Header{
		Height:       last.Height + 1,
		Timestamp:    time.Now().UnixNano() / 1e6,
		TotalFee:     0,
		PreviousHash: parentHash,
		Coinbase:     coinbase,
		StatTree:     MPTPlus.MTP_Tree(db.GetDBInst(), last.StatTree.Root),
		TokenTree:    MPTPlus.MTP_Tree(db.GetDBInst(), last.TokenTree.Root),
		Version:      HEADER_VERSION_MIXED,
	}

	return block
}

func (header *Header) NewSubTransaction(tx userevent.SubTransactions) bool {
	// TODO sub transaction
	return true
}

func (header *Header) CheckAmount(tx userevent.Transaction) (*types.Account, bool) {
	account, err := header.GetAccount(tx.GetFrom())
	if err != nil || account == nil || account.Gas < tx.Fee {
		return account, false
	}
	switch tx.TokenAddress {
	case types.EKTAddress:
		if account.Amount < tx.Amount {
			return account, false
		}
	case types.GasAddress:
		if account.Gas < tx.Amount+tx.Fee {
			return account, false
		}
	default:
		if account.Balances == nil || account.Balances[tx.TokenAddress] < tx.Amount {
			return account, false
		}
	}
	account.BurnGas(tx.Fee)
	header.StatTree.MustInsert(account.Address, account.ToBytes())
	return account, true
}

func (header *Header) CheckContractTransfer(tx userevent.Transaction) bool {
	if _, success := header.CheckAmount(tx); !success {
		return false
		//} else {
		//	address, sub := tx.GetTo()[:32], tx.GetTo()[32:]
		//	account, err := header.GetAccount(address)
		//	if err != nil || account == nil {
		//		if hex.EncodeToString(address) == contract.SYSTEM_AUTHOR {
		//			account = types.NewAccount(address)
		//		}
		//	}
		//	if account != nil {
		//		if account.Contracts == nil {
		//			account.Contracts = make(map[string]types.ContractAccount)
		//		}
		//		c, exist := account.Contracts[hex.EncodeToString(sub)]
		//		if !exist {
		//			contractAccount := types.NewContractAccount(sub, nil)
		//			c = *contractAccount
		//		}
		//		account.Contracts[hex.EncodeToString(sub)] = c
		//		header.StatTree.MustInsert(account.Address, account.ToBytes())
		//	}
	}

	return true
}

func (header *Header) NewTransaction(tx userevent.Transaction) userevent.TransactionReceipt {
	account, success := header.CheckAmount(tx)
	if !success {
		return userevent.NewTransactionReceipt(tx, false, userevent.FailType_NO_ENOUGH_AMOUNT)
	}

	header.TotalFee += tx.Fee

	receiverAccount, err := header.GetAccount(tx.GetTo())
	if receiverAccount == nil || err != nil {
		receiverAccount = types.NewAccount(tx.GetTo())
	}

	if tx.Nonce != account.Nonce+1 {
		return userevent.NewTransactionReceipt(tx, false, userevent.FailType_Invalid_NONCE)
	} else if tx.TokenAddress == "" {
		if account.GetAmount() < tx.Amount {
			return userevent.NewTransactionReceipt(tx, false, userevent.FailType_NO_ENOUGH_AMOUNT)
		} else {
			account.ReduceAmount(tx.Amount)
			receiverAccount.AddAmount(tx.Amount)
			header.StatTree.MustInsert(tx.GetFrom(), account.ToBytes())
			header.StatTree.MustInsert(tx.GetTo(), receiverAccount.ToBytes())
			return userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
		}
	} else {
		if account.Balances[tx.TokenAddress] < tx.Amount {
			return userevent.NewTransactionReceipt(tx, false, userevent.FailType_NO_ENOUGH_AMOUNT)
		} else {
			account.Balances[tx.TokenAddress] -= tx.Amount
			account.Nonce++
			if receiverAccount.Balances == nil {
				receiverAccount.Balances = make(map[string]int64)
				receiverAccount.Balances[tx.TokenAddress] = 0
			}
			receiverAccount.Balances[tx.TokenAddress] += tx.Amount
			header.StatTree.MustInsert(tx.GetFrom(), account.ToBytes())
			header.StatTree.MustInsert(tx.GetTo(), receiverAccount.ToBytes())
			return userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
		}
	}
}

func FromBytes2Header(data []byte) *Header {
	var header Header
	err := json.Unmarshal(data, &header)
	if err != nil {
		return nil
	}
	return &header
}

func (header Header) ValidateBlockStat(next Header, transactions []userevent.Transaction, receipts userevent.Receipts) bool {
	log.Info("Validating header stat merkler proof.")

	//根据上一个区块头生成一个新的区块
	_next := NewHeader(header, header.CaculateHash(), next.Coinbase)

	//让新生成的区块执行peer传过来的body中的user events进行计算
	if len(transactions) > 0 {
		for i, transaction := range transactions {
			receipt := receipts[i]
			if receipt.Fee != transaction.Fee {
				transaction.Fee = receipt.Fee
			}
			_receipt := _next.NewTransaction(transaction)
			if !receipt.EqualsTo(_receipt) {
				return false
			}
		}
	}

	_next.UpdateMiner()

	// 判断默克尔根是否相同
	if !bytes.Equal(next.StatTree.Root, _next.StatTree.Root) || !bytes.Equal(next.TokenTree.Root, _next.TokenTree.Root) {
		return false
	}

	return true
}

func (header *Header) UpdateMiner() {
	account, err := header.GetAccount(header.Coinbase)
	if account == nil || err != nil {
		account = types.NewAccount(header.Coinbase)
	}
	account.Gas += header.TotalFee
	err = header.StatTree.MustInsert(header.Coinbase, account.ToBytes())
	if err != nil {
		log.Crit("Update miner failed, %s", err.Error())
	}
}

func Decimals(decimal int64) int64 {
	result := int64(1)
	for i := int64(0); i < decimal; i++ {
		result *= 10
	}
	return result
}
