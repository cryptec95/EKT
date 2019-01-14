package pool

import (
	"bytes"
	"encoding/hex"

	"github.com/EducationEKT/EKT/MPTPlus"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
)

type TxPool struct {
	All      *TransactionDict `json:"All"`
	List     *TxTimedList     `json:"timedList"`
	UsersTxs *UsersTxs        `json:"userTxs"`
}

func NewTxPool() *TxPool {
	pool := &TxPool{
		All:      NewTransactionDict(),
		List:     NewTimedList(),
		UsersTxs: NewUsersTxs(),
	}

	return pool
}

func (pool *TxPool) Park(tx *userevent.Transaction, userNonce int64) {
	if pool.All.Get(tx.TransactionId()) == nil {
		pool.All.Save(tx)
		if txs, ready := pool.UsersTxs.SaveTx(tx, userNonce); ready {
			pool.List.Put(txs...)
		}
	}
}

func (pool *TxPool) GetTx(hash []byte) *userevent.Transaction {
	return pool.All.Get(hex.EncodeToString(hash))
}

func (pool *TxPool) Pop(size int) []*userevent.Transaction {
	txs := pool.List.Pop(size)
	return txs
}

func (pool *TxPool) Notify(txs []userevent.Transaction) {
	for _, tx := range txs {
		pool.All.Delete(tx.TransactionId())
		pool.UsersTxs.Notify(tx)
		pool.List.Notify(tx)
	}
}

func (pool *TxPool) GetUserTxs(address string) *UserTxs {
	pool.UsersTxs.locker.RLock()
	result := pool.UsersTxs.M[address]
	pool.UsersTxs.locker.RUnlock()
	return result
}

func (pool *TxPool) Promote(statdb MPTPlus.MTP) {
	pool.UsersTxs.locker.Lock()
	for addr, userTxs := range pool.UsersTxs.M {
		var account types.Account
		address, _ := hex.DecodeString(addr)
		err := statdb.GetInterfaceValue(address, &account)
		if err == nil && bytes.Equal(address, account.Address) {
			txs, exist := userTxs.Promote(address, account.Nonce)
			if exist {
				pool.List.Put(txs...)
			}
		}
		pool.UsersTxs.M[addr] = userTxs
	}
	pool.UsersTxs.locker.Unlock()
}
