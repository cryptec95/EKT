package contract

import (
	"encoding/hex"
	"encoding/json"

	"github.com/EducationEKT/EKT/context"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/vm"
)

func Run(tx userevent.Transaction, account *types.Account) (*userevent.TransactionReceipt, []byte) {
	c := getContract(tx.To, account)
	contractData := c.Data()
	if c == nil {
		return userevent.ContractRefuseTx(tx), nil
	}
	receipt, data := c.Call(tx)
	if receipt == nil {
		receipt = userevent.ContractRefuseTx(tx)
	}
	if receipt.Success {
		contractData = data
	}
	c.Recover(contractData)
	updateContract(tx.To, c)
	return receipt, data
}

func InitContractAccount(tx userevent.Transaction, account *types.Account) bool {
	switch hex.EncodeToString(tx.To[:32]) {
	case SYSTEM_AUTHOR:
		switch hex.EncodeToString(tx.To[32:]) {
		case EKT_GAS_BANCOR_CONTRACT:
			contract := types.NewContractAccount(tx.To[32:], nil)
			contract.Gas = 1e8
			if account.Contracts == nil {
				account.Contracts = make(map[string]types.ContractAccount)
			}
			account.Contracts[hex.EncodeToString(tx.To[32:])] = *contract
			return true
		}
	}
	return false
}

func InitContract(sticker *context.Sticker, account *types.Account, tx userevent.Transaction) (types.HexBytes, types.HexBytes, error) {
	previousHash, _ := sticker.GetBytes("previousHash")
	timestamp, _ := sticker.GetInt64("timestamp")

	contractHash := crypto.Sha3_256([]byte(tx.Data))
	db.GetDBInst().Set(contractHash, []byte(tx.Data))

	evm := vm.NewVM(previousHash, timestamp)
	_, err := evm.Run(tx.Data)
	if err != nil {
		return nil, contractHash, err
	}

	value, err := evm.Run(`
		init();
		return JSON.stringify({
			"prop": prop;
			"contract": contract
		});
	`)
	if err != nil {
		return nil, contractHash, err
	}
	var result ContractData
	err = json.Unmarshal([]byte(value.String()), &result)
	if err != nil {
		return nil, contractHash, err
	}
	return result.Bytes(), contractHash, nil
}
