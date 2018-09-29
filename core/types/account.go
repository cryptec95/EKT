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

func (account *Account) AddAmount(amount int64) {
	account.Amount = account.Amount + amount
}

func (account *Account) ReduceAmount(amount int64) {
	account.Amount -= amount
	account.Nonce++
}

func (account *Account) BurnGas(gas int64) {
	account.Gas = account.Gas - gas
}

func FromPubKeyToAddress(pubKey []byte) []byte {
	hash := crypto.Sha3_256(pubKey)
	address := crypto.Sha3_256(crypto.Sha3_256(append([]byte("EKT"), hash...)))
	return address
}
