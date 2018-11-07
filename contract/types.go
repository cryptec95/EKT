package contract

import (
	"encoding/hex"
	"encoding/json"
	"github.com/EducationEKT/EKT/bancor"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"math"
)

type ContractData struct {
	Prop     map[string]interface{} `json:"prop"`
	Contract map[string]interface{} `json:"contract"`
}

func (data ContractData) Bytes() []byte {
	result, _ := json.Marshal(data)
	return result
}

var contracts Contracts

func init() {
	contracts = Contracts{
		m: make(map[string]Contract),
	}
}

type Contract interface {
	Recover(data []byte) bool
	Data() []byte
	Call(tx userevent.Transaction) (*userevent.TransactionReceipt, []byte)
}

type Contracts struct {
	m map[string]Contract
}

func getContract(address []byte, account *types.Account) Contract {
	return initContract(address, account)
}

func updateContract(address []byte, c Contract) {
	contracts.m[hex.EncodeToString(address)] = c
}

func initContract(address []byte, account *types.Account) Contract {
	author, subAddress := address[:32], address[32:]
	switch hex.EncodeToString(author) {
	case SYSTEM_AUTHOR:
		switch hex.EncodeToString(subAddress) {
		case EKT_GAS_BANCOR_CONTRACT:
			b := bancor.NewBancor(EKT_GAS_PARAM_CW, math.Pow10(types.EKT_DECIMAL), EKT_GAS_PARAM_INIT_SMART_TOKEN, EKT_GAS_PARAM_TOTAL_SMART_TOKEN, types.EKTAddress, types.GasAddress)
			contractAccount := account.Contracts[hex.EncodeToString(subAddress)]
			if contractAccount.ContractData != nil {
				b.Recover(contractAccount.ContractData)
			}
			return b
		default:
		}
	default:
	}
	return nil
}
