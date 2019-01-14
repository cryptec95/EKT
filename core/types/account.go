package types

import (
	"encoding/json"

	"github.com/EducationEKT/EKT/crypto"
)

type Account struct {
	Address   HexBytes                   `json:"address"`
	Amount    int64                      `json:"amount"`
	Gas       int64                      `json:"gas"`
	Nonce     int64                      `json:"nonce"`
	Contracts map[string]ContractAccount `json:"contracts"`
	Balances  map[string]int64           `json:"balances"`
}

type AccountChange struct {
	M map[string]int64
}

func NewAccountChange() *AccountChange {
	return &AccountChange{
		M: make(map[string]int64),
	}
}

func (change *AccountChange) Add(tokenAddress string, amount int64) {
	value, exist := change.M[tokenAddress]
	if !exist {
		value = 0
	}
	value = value + amount
	change.M[tokenAddress] = value
}

func (change *AccountChange) Reduce(tokenAddress string, amount int64) {
	value, exist := change.M[tokenAddress]
	if !exist {
		value = 0
	}
	value = value - amount
	change.M[tokenAddress] = value
}

func NewAccount(address []byte) *Account {
	return &Account{
		Address:   address,
		Nonce:     0,
		Gas:       0,
		Amount:    0,
		Balances:  make(map[string]int64),
		Contracts: make(map[string]ContractAccount),
	}
}

func (account Account) ToBytes() []byte {
	data, _ := json.Marshal(account)
	return data
}

func (account Account) GetNonce() int64 {
	return account.Nonce
}

func (account Account) GetAmount() int64 {
	return account.Amount
}

func (account *Account) BurnGas(gas int64) {
	account.Gas = account.Gas - gas
	account.Nonce++
}

func FromPubKeyToAddress(pubKey []byte) []byte {
	hash := crypto.Sha3_256(pubKey)
	address := crypto.Sha3_256(crypto.Sha3_256(append([]byte("EKT"), hash...)))
	return address
}

func (account *Account) Transfer(change AccountChange) bool {
	for tokenAddr, amount := range change.M {
		switch tokenAddr {
		case EKTAddress:
			account.Amount += amount
			if account.Amount < 0 {
				return false
			}
		case GasAddress:
			account.Gas += amount
			if account.Gas < 0 {
				return false
			}
		default:
			if account.Balances == nil {
				account.Balances = make(map[string]int64)
			}
			count, exist := account.Balances[tokenAddr]
			if !exist {
				count = 0
			}
			count += amount
			if count < 0 {
				return false
			}
			account.Balances[tokenAddr] = count
		}
	}
	return true
}
