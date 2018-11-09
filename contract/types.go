package contract

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	"github.com/EducationEKT/EKT/bancor"
	"github.com/EducationEKT/EKT/context"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/vm"
)

type ContractData struct {
	Prop     map[string]interface{} `json:"prop"`
	Contract map[string]interface{} `json:"contract"`
}

func (data ContractData) Bytes() []byte {
	result, _ := json.Marshal(data)
	return result
}

type IContract interface {
	Recover(data []byte) bool
	Data() []byte
	Call(tx userevent.Transaction) (*userevent.TransactionReceipt, []byte)
}

type VMContract struct {
	VM *vm.Otto
}

func NewVMContract(lastHash []byte, timestamp int64) *VMContract {
	return &VMContract{
		VM: vm.NewVM(lastHash, timestamp),
	}
}

func (vmContract VMContract) Recover(data []byte) bool {
	_, _, err := vm.Run("contract = JSON.parse(" + string(data) + ").contract;")
	if err != nil {
		return false
	}
	return true
}

func (vmContract VMContract) Data() []byte {
	_, value, err := vm.Run(`
		var data = JSON.stringify(contract);
		return data;
	`)
	if err != nil {
		return nil
	}
	return []byte(value.String())
}

func (vmContract VMContract) Call(tx userevent.Transaction) (*userevent.TransactionReceipt, []byte) {
	call := fmt.Sprintf(`
		var result = call(%s);
		return JSON.stringify(result);
	`)
	_, value, err := vm.Run(call)
	if err != nil {
		return userevent.ContractRefuseTx(tx), nil
	}
	var subTxs []userevent.SubTransaction
	err = json.Unmarshal([]byte(value.String()), &subTxs)
	if err != nil {
		return userevent.ContractRefuseTx(tx), nil
	}
	receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
	receipt.SubTransactions = subTxs
	return &receipt, vmContract.Data()
}

func getContract(ctx *context.Sticker, address []byte, account *types.Account) IContract {
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
		if account.Contracts == nil {
			return nil
		} else if contractAccount, exist := account.Contracts[hex.EncodeToString(subAddress)]; !exist {
			return nil
		} else {
			contract, err := db.GetDBInst().Get(contractAccount.CodeHash)
			if err != nil {
				return nil
			}
			contractData := string(contractAccount.ContractData)
			lastHash, _ := ctx.GetBytes("lastHash")
			timestamp, _ := ctx.GetInt64("timestamp")
			vmContract := NewVMContract(lastHash, timestamp)
			_, err = vmContract.VM.Run(contract)
			if err != nil {
				return nil
			}
			if !vmContract.Recover([]byte(contractData)) {
				return nil
			}
			return vmContract
		}
	}
	return nil
}
