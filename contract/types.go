package contract

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/bancor"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"sync"
)

var contracts Contracts

func init() {
	contracts = Contracts{
		m:      make(map[string]Contract),
		locker: sync.RWMutex{},
	}
}

type Contract interface {
	Recover(data []byte) bool
	Data() []byte
	Call(tx userevent.Transaction) (*userevent.TransactionReceipt, []byte)
}

type Contracts struct {
	m      map[string]Contract
	locker sync.RWMutex
}

func GetContract(address []byte, account *types.Account) Contract {
	contracts.locker.RLock()
	contract, inited := contracts.m[hex.EncodeToString(address)]
	contracts.locker.RUnlock()

	subAddress := address[32:]
	if !inited {
		if account.Contracts != nil {
			c, exist := account.Contracts[hex.EncodeToString(subAddress)]
			if !exist {
				contract = initContract(address)
			} else {
				contract.Recover(c.ContractData)
			}
		} else {
			contract = initContract(address)
		}
	}

	return contract
}

func initContract(address []byte) Contract {
	author, subAddress := address[:32], address[32:]
	switch hex.EncodeToString(author) {
	case SYSTEM_AUTHOR:
		switch hex.EncodeToString(subAddress) {
		case EKT_GAS_BANCOR_CONTRACT:
			return bancor.NewBancor(EKT_GAS_PARAM_CW, EKT_GAS_PARAM_INIT_CONNECT_TOKEN, EKT_GAS_PARAM_INIT_SMART_TOKEN, types.EKTAddress, types.GasAddress)
		default:
			// TODO other contract not open now
		}
	default:
		// TODO user contract not open now
	}
	return nil
}
