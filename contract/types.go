package contract

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/bancor"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"math"
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

func getContract(address []byte, account *types.Account) Contract {
	contracts.locker.RLock()
	contract, exist := contracts.m[hex.EncodeToString(address)]
	contracts.locker.RUnlock()

	if !exist {
		contract = initContract(address, account)
	}

	return contract
}

func updateContract(address []byte, c Contract) {
	contracts.locker.Lock()
	contracts.m[hex.EncodeToString(address)] = c
	contracts.locker.Unlock()
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
