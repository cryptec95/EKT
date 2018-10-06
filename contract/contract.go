package contract

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
)

func Run(tx userevent.Transaction, account *types.Account) (*userevent.TransactionReceipt, []byte) {
	c := getContract(tx.To, account)
	if c == nil {
		return userevent.ContractRefuseTx(tx), nil
	}
	receipt, data := c.Call(tx)
	if receipt == nil {
		receipt = userevent.ContractRefuseTx(tx)
	}
	if receipt.Success {
		updateContract(tx.To, c)
	}
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
