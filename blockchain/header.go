package blockchain

import (
	"encoding/hex"
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

func (header *Header) NewSubTransaction(txs userevent.SubTransactions) bool {
	m := make(map[string]*types.Account)
	for _, tx := range txs {
		from, exist1 := m[hex.EncodeToString(tx.From[:32])]
		to, exist2 := m[hex.EncodeToString(tx.To[:32])]
		if !exist1 {
			_from, err := header.GetAccount(tx.From[:32])
			if err != nil {
				return false
			}
			from = _from
		}
		if !exist2 {
			_to, err := header.GetAccount(tx.To[:32])
			if err != nil {
				_to = types.NewAccount(tx.To[:32])
			}
			if _, exist := _to.Contracts[hex.EncodeToString(tx.To[32:])]; !exist {
				log.Crit("no such contract address")
				return false
			}
			to = _to
		}
		if !header.HandleTx(from, to, tx) {
			return false
		} else {
			m[hex.EncodeToString(tx.From[:32])] = from
			m[hex.EncodeToString(tx.To[:32])] = to
		}
	}
	for _, account := range m {
		header.StatTree.MustInsert(account.Address, account.ToBytes())
	}
	return true
}

func (header *Header) HandleTx(from, to *types.Account, tx userevent.SubTransaction) bool {
	if !header.CheckSubTx(from, to, tx) {
		return false
	} else {
		return header.Transfer(from, to, tx)
	}
}

func (header *Header) Transfer(from, to *types.Account, tx userevent.SubTransaction) bool {
	if len(tx.From) == 32 {
		switch tx.TokenAddress {
		case types.EKTAddress:
			from.Amount -= tx.Amount
		case types.GasAddress:
			from.Gas -= tx.Amount
		default:
			from.Balances[tx.TokenAddress] -= tx.Amount
		}
	} else {
		contractAccount := from.Contracts[hex.EncodeToString(tx.From[32:])]
		switch tx.TokenAddress {
		case types.EKTAddress:
			contractAccount.Amount -= tx.Amount
		case types.GasAddress:
			contractAccount.Gas -= tx.Amount
		default:
			contractAccount.Balances[tx.TokenAddress] -= tx.Amount
		}
		from.Contracts[hex.EncodeToString(tx.From[32:])] = contractAccount
	}

	if len(tx.To) == 32 {
		switch tx.TokenAddress {
		case types.EKTAddress:
			to.Amount += tx.Amount
		case types.GasAddress:
			to.Gas += tx.Amount
		default:
			if to.Balances == nil {
				to.Balances = map[string]int64{}
				to.Balances[tx.TokenAddress] = 0
			}
			to.Balances[tx.TokenAddress] += tx.Amount
		}
	} else {
		contractAccount := to.Contracts[hex.EncodeToString(tx.To[32:])]
		switch tx.TokenAddress {
		case types.EKTAddress:
			contractAccount.Amount += tx.Amount
		case types.GasAddress:
			contractAccount.Gas += tx.Amount
		default:
			if contractAccount.Balances == nil {
				contractAccount.Balances = map[string]int64{}
				contractAccount.Balances[tx.TokenAddress] = 0
			}
			contractAccount.Balances[tx.TokenAddress] += tx.Amount
		}
		to.Contracts[hex.EncodeToString(tx.To[32:])] = contractAccount
	}
	return true
}

func (header *Header) CheckSubTx(from, to *types.Account, tx userevent.SubTransaction) bool {
	if len(tx.From) == 32 {
		switch tx.TokenAddress {
		case types.EKTAddress:
			return from.Amount > tx.Amount
		case types.GasAddress:
			return from.Gas > tx.Amount
		default:
			if from.Balances != nil && from.Balances[tx.TokenAddress] > tx.Amount {
				return true
			}
		}
	} else if from.Contracts == nil {
		return false
	} else {
		subAddr := tx.From[32:]
		contractAccount := from.Contracts[hex.EncodeToString(subAddr)]
		switch tx.TokenAddress {
		case types.EKTAddress:
			return contractAccount.Amount > tx.Amount
		case types.GasAddress:
			return contractAccount.Gas > tx.Amount
		default:
			if contractAccount.Balances != nil && contractAccount.Balances[tx.TokenAddress] > tx.Amount {
				return true
			}
		}
	}
	return false
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

//func (header *Header) NewTransaction(tx userevent.Transaction) userevent.TransactionReceipt {
//	account, success := header.CheckAmount(tx)
//	if !success {
//		return userevent.NewTransactionReceipt(tx, false, userevent.FailType_NO_ENOUGH_AMOUNT)
//	}
//
//	header.TotalFee += tx.Fee
//
//	receiverAccount, err := header.GetAccount(tx.GetTo())
//	if receiverAccount == nil || err != nil {
//		receiverAccount = types.NewAccount(tx.GetTo())
//	}
//
//	if tx.Nonce != account.Nonce+1 {
//		return userevent.NewTransactionReceipt(tx, false, userevent.FailType_Invalid_NONCE)
//	} else if tx.TokenAddress == "" {
//		if account.GetAmount() < tx.Amount {
//			return userevent.NewTransactionReceipt(tx, false, userevent.FailType_NO_ENOUGH_AMOUNT)
//		} else {
//			account.ReduceAmount(tx.Amount)
//			receiverAccount.AddAmount(tx.Amount)
//			header.StatTree.MustInsert(tx.GetFrom(), account.ToBytes())
//			header.StatTree.MustInsert(tx.GetTo(), receiverAccount.ToBytes())
//			return userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
//		}
//	} else {
//		if account.Balances[tx.TokenAddress] < tx.Amount {
//			return userevent.NewTransactionReceipt(tx, false, userevent.FailType_NO_ENOUGH_AMOUNT)
//		} else {
//			account.Balances[tx.TokenAddress] -= tx.Amount
//			account.Nonce++
//			if receiverAccount.Balances == nil {
//				receiverAccount.Balances = make(map[string]int64)
//				receiverAccount.Balances[tx.TokenAddress] = 0
//			}
//			receiverAccount.Balances[tx.TokenAddress] += tx.Amount
//			header.StatTree.MustInsert(tx.GetFrom(), account.ToBytes())
//			header.StatTree.MustInsert(tx.GetTo(), receiverAccount.ToBytes())
//			return userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
//		}
//	}
//}

func FromBytes2Header(data []byte) *Header {
	var header Header
	err := json.Unmarshal(data, &header)
	if err != nil {
		return nil
	}
	return &header
}

//func (header Header) ValidateBlockStat(next Header, transactions []userevent.Transaction, receipts userevent.Receipts) bool {
//	log.Info("Validating header stat merkler proof.")
//
//	//根据上一个区块头生成一个新的区块
//	_next := NewHeader(header, header.CaculateHash(), next.Coinbase)
//
//	//让新生成的区块执行peer传过来的body中的user events进行计算
//	if len(transactions) > 0 {
//		for i, transaction := range transactions {
//			receipt := receipts[i]
//			if receipt.Fee != transaction.Fee {
//				transaction.Fee = receipt.Fee
//			}
//			_receipt := _next.NewTransaction(transaction)
//			if !receipt.EqualsTo(_receipt) {
//				return false
//			}
//		}
//	}
//
//	_next.UpdateMiner()
//
//	// 判断默克尔根是否相同
//	if !bytes.Equal(next.StatTree.Root, _next.StatTree.Root) || !bytes.Equal(next.TokenTree.Root, _next.TokenTree.Root) {
//		return false
//	}
//
//	return true
//}

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
