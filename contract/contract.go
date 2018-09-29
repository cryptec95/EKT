package contract

import (
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
)

func Run(tx userevent.Transaction, account *types.Account) (*userevent.TransactionReceipt, []byte) {
	c := GetContract(tx.To, account)
	if c == nil {
		return userevent.ContractRefuseTx(tx), nil
	}
	receipt, data := c.Call(tx)
	if receipt == nil {
		receipt = userevent.ContractRefuseTx(tx)
	}
	return receipt, data
}
