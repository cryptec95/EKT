package pool

import (
	"github.com/EducationEKT/EKT/core/userevent"
)

type TxPool struct {
	all      *TransactionDict
	list     *TxTimedList
	usersTxs *UsersTxs
}

func NewTxPool() *TxPool {
	pool := &TxPool{
		all:      NewTransactionDict(),
		list:     NewTimedList(),
		usersTxs: NewUsersTxs(),
	}

	return pool
}

func (pool *TxPool) Park(tx *userevent.Transaction, userNonce int64) {
	if pool.all.Get(tx.TransactionId()) == nil {
		pool.all.Save(tx)
		if pool.usersTxs.SaveTx(tx, userNonce) {
			pool.list.Put(tx)
		}
	}
}

func (pool *TxPool) Pop(size int) []*userevent.Transaction {
	return pool.list.Pop(size)
}

func (pool *TxPool) Notify(txs []userevent.Transaction) {
	for _, tx := range txs {
		pool.all.Delete(tx.TransactionId())
		pool.usersTxs.Remove(tx)
		pool.list.Notify(tx)
	}
}

func (pool *TxPool) GetUserTxs(address string) *UserTxs {
	return pool.usersTxs.m[address]
}
