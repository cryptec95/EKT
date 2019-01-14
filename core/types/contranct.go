package types

import (
	"encoding/json"
	"fmt"
)

// Contract address contains 64 bytes
// The first 32 byte represents the founders of contract, it is create by system if the first
// The end 32 byte represents the true address of contract for a founder
type ContractProp struct {
	Name       string `json:"name"`
	Author     string `json:"author"`
	Upgradable bool   `json:"upgradable"`
}

//func (contractProp *ContractProp) UnmarshalJSON(data []byte) error {
//	data = bytes.Trim(data, `"`)
//	var _contractProp ContractProp
//	err := json.Unmarshal(data, &_contractProp)
//	*contractProp = _contractProp
//	return err
//}

func (contractProp *ContractProp) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(contractProp)
	return []byte(fmt.Sprintf(`"%s"`, string(data))), err
}

type ContractData struct {
	Prop     ContractProp `json:"prop"`
	Contract string       `json:"contract"`
}

//func (contractData *ContractData) UnmarshalJSON(data []byte) error {
//	data = bytes.Trim(data, `"`)
//	var _contractData ContractData
//	err := json.Unmarshal(data, &_contractData)
//	*contractData = _contractData
//	return err
//}

func (contractData *ContractData) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(contractData)
	return []byte(fmt.Sprintf(`"%s"`, string(data))), err
}

func (contractData ContractData) Bytes() []byte {
	result, _ := json.Marshal(contractData)
	return result
}

type ContractAccount struct {
	Address      HexBytes         `json:"address"`
	Amount       int64            `json:"amount"`
	Gas          int64            `json:"gas"`
	CodeHash     HexBytes         `json:"codeHash"`
	ContractData ContractData     `json:"data"`
	Balances     map[string]int64 `json:"balances"`
}

func NewContractAccount(address []byte, contractHash []byte, contractData ContractData) *ContractAccount {
	return &ContractAccount{
		Address:      address,
		Amount:       0,
		Gas:          0,
		Balances:     make(map[string]int64),
		CodeHash:     contractHash,
		ContractData: contractData,
	}
}

func (account *ContractAccount) Transfer(change AccountChange) bool {
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
