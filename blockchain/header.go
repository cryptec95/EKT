package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

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
	HEADER_VERSION_TOPIC    = 2
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
	TxRoot       *MPTPlus.MTP   `json:"txRoot"`
	ReceiptHash  types.HexBytes `json:"receiptHash"`
	ReceiptRoot  *MPTPlus.MTP   `json:"receiptRoot"`

	ChainStat  *MPTPlus.MTP `json:"chainStat"`
	ChainEvent *MPTPlus.MTP `json:"chainEvent"`

	Version int `json:"version"`
}

func (header *Header) Bytes() []byte {
	data, _ := json.Marshal(header)
	return data
}

func (header *Header) CalculateHash() []byte {
	if header.Version == HEADER_VERSION_MIXED {
		data := fmt.Sprintf(`{"height":%d,"timestamp":%d,"totalFee":%d,"previousHash":"%s","miner":"%s","statRoot":"%s","tokenRoot":"%s","txHash":"%s","receiptHash":"%s","version":%d}`,
			header.Height, header.Timestamp, header.TotalFee, hex.EncodeToString(header.PreviousHash), hex.EncodeToString(header.Coinbase), hex.EncodeToString(header.StatTree.Root),
			hex.EncodeToString(header.TokenTree.Root), hex.EncodeToString(header.TxHash), hex.EncodeToString(header.ReceiptHash), header.Version)
		return crypto.Sha3_256([]byte(data))
	} else {
		return crypto.Sha3_256(header.Bytes())
	}
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

func NewHeader(last Header, packTime int64, parentHash types.HexBytes, coinbase types.HexBytes) *Header {
	header := &Header{
		Height:       last.Height + 1,
		Timestamp:    packTime,
		TotalFee:     0,
		PreviousHash: parentHash,
		Coinbase:     coinbase,
		StatTree:     MPTPlus.MTP_Tree(db.GetDBInst(), last.StatTree.Root),
		TokenTree:    MPTPlus.MTP_Tree(db.GetDBInst(), last.TokenTree.Root),
		Version:      HEADER_VERSION_MIXED,
	}

	return header
}

func (header *Header) NewSubTransaction(txs userevent.SubTransactions) bool {
	changes := make(map[string]*types.AccountChange)

	for _, tx := range txs {
		from, exist := changes[hex.EncodeToString(tx.From)]
		if !exist {
			from = types.NewAccountChange()
		}
		from.Reduce(tx.TokenAddress, tx.Amount)
		changes[hex.EncodeToString(tx.From)] = from

		to, exist := changes[hex.EncodeToString(tx.To)]
		if !exist {
			to = types.NewAccountChange()
		}
		to.Add(tx.TokenAddress, tx.Amount)
		changes[hex.EncodeToString(tx.To)] = to
	}

	for addr, change := range changes {
		address, err := hex.DecodeString(addr)
		if err != nil {
			return false
		}
		if len(address) == types.AccountAddressLength {
			account, err := header.GetAccount(address)
			if err != nil || account == nil {
				account = types.NewAccount(address)
			}
			if !account.Transfer(*change) {
				return false
			}
			header.StatTree.MustInsert(account.Address, account.ToBytes())
		} else if len(address) == types.ContractAddressLength {
			account, err := header.GetAccount(address[:32])
			if err != nil || account == nil || account.Contracts == nil {
				return false
			}
			contractAccount, exist := account.Contracts[hex.EncodeToString(address[32:])]
			if !exist {
				return false
			}
			if !contractAccount.Transfer(*change) {
				return false
			}
			account.Contracts[hex.EncodeToString(address[32:])] = contractAccount
			header.StatTree.MustInsert(account.Address, account.ToBytes())
		} else {
			return false
		}
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
	if bytes.EqualFold(tx.From, tx.To) {
		return false
	}
	if len(tx.From) == 32 {
		switch tx.TokenAddress {
		case types.EKTAddress:
			return from.Amount >= tx.Amount
		case types.GasAddress:
			return from.Gas >= tx.Amount
		default:
			if from.Balances != nil && from.Balances[tx.TokenAddress] >= tx.Amount {
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
			return contractAccount.Amount >= tx.Amount
		case types.GasAddress:
			return contractAccount.Gas >= tx.Amount
		default:
			if contractAccount.Balances != nil && contractAccount.Balances[tx.TokenAddress] >= tx.Amount {
				return true
			}
		}
	}
	return false
}

func (header *Header) CheckFromAndBurnGas(tx userevent.Transaction) bool {
	if len(tx.From) != types.AccountAddressLength {
		return false
	}

	if len(tx.To) != 0 && len(tx.To) != types.AccountAddressLength && len(tx.To) != types.ContractAddressLength {
		return false
	}
	if tx.Amount < 0 {
		return false
	}
	account, err := header.GetAccount(tx.GetFrom())
	if err != nil || account == nil {
		account = types.NewAccount(tx.From)
	}
	if account.Gas < tx.Fee || account.GetNonce()+1 != tx.GetNonce() {
		return false
	}
	switch tx.TokenAddress {
	case types.EKTAddress:
		if account.Amount < tx.Amount {
			return false
		}
	case types.GasAddress:
		if account.Gas < tx.Amount+tx.Fee {
			return false
		}
	default:
		if account.Balances == nil || account.Balances[tx.TokenAddress] < tx.Amount {
			return false
		}
	}
	account.BurnGas(tx.Fee)
	header.StatTree.MustInsert(account.Address, account.ToBytes())
	return true
}

func (header *Header) CheckTransfer(tx userevent.Transaction) bool {
	return header.CheckFromAndBurnGas(tx)
}

func FromBytes2Header(data []byte) *Header {
	var header Header
	err := json.Unmarshal(data, &header)
	if err != nil {
		return nil
	}
	return &header
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
