package contract

import (
	"encoding/hex"
	"encoding/json"
	"math"

	"github.com/EducationEKT/EKT/bancor"
	"github.com/EducationEKT/EKT/context"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/vm"
)

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
	vmContract.VM.Set("data", string(data))
	_, err := vmContract.VM.Run("contract = JSON.parse(data);")
	if err != nil {
		return false
	}
	return true
}

func (vmContract VMContract) Data() []byte {
	_, err := vmContract.VM.Run(`
		var data = JSON.stringify(contract);
	`)
	if err != nil {
		return nil
	}
	value, err := vmContract.VM.Get("data")
	if err != nil {
		return nil
	}
	return []byte(value.String())
}

func (vmContract VMContract) Call(tx userevent.Transaction) (*userevent.TransactionReceipt, []byte) {
	vmContract.VM.Set("data", tx.Data)
	vmContract.VM.Set("additional", tx.Additional)
	vmContract.VM.Set("tx", string(tx.Bytes()))
	call := `
		var transaction = JSON.parse(tx);
		var result = call();
		var txs = "[]";
		if (result !== undefined && result !== null) {
			txs = JSON.stringify(result);
		}
	`
	_, err := vmContract.VM.Run(call)
	if err != nil {
		return userevent.ContractRefuseTx(tx), nil
	}
	value, err := vmContract.VM.Get("txs")
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
			if len(contractAccount.ContractData) > 0 {
				var contractData types.ContractData
				json.Unmarshal(contractAccount.ContractData, &contractData)
				b.Recover([]byte(contractData.Contract))
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
			lastHash, _ := ctx.GetBytes("lastHash")
			timestamp, _ := ctx.GetInt64("timestamp")
			vmContract := NewVMContract(lastHash, timestamp)
			_, err = vmContract.VM.Run(string(contract))
			if err != nil {
				return nil
			}
			var contractData types.ContractData
			json.Unmarshal(contractAccount.ContractData, &contractData)
			if !vmContract.Recover([]byte(contractData.Contract)) {
				return nil
			}
			return vmContract
		}
	}
	return nil
}
