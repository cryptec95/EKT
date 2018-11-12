package types

import (
	"encoding/json"
)

// Contract address contains 64 bytes
// The first 32 byte represents the founders of contract, it is create by system if the first
// The end 32 byte represents the true address of contract for a founder

type ContractData struct {
	Prop     string `json:"prop"`
	Contract string `json:"contract"`
}

func (data ContractData) Bytes() []byte {
	result, _ := json.Marshal(data)
	return result
}

type ContractAccount struct {
	Address      HexBytes         `json:"address"`
	Amount       int64            `json:"amount"`
	Gas          int64            `json:"gas"`
	CodeHash     HexBytes         `json:"codeHash"`
	ContractData []byte           `json:"data"`
	Balances     map[string]int64 `json:"balances"`
}

func NewContractAccount(address []byte, contractHash []byte, contractData ContractData) *ContractAccount {
	return &ContractAccount{
		Address:      address,
		Amount:       0,
		Gas:          0,
		Balances:     make(map[string]int64),
		CodeHash:     contractHash,
		ContractData: contractData.Bytes(),
	}
}

func (account ContractAccount) Transfer(change AccountChange) bool {
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
