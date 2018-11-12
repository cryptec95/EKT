package pool

import (
	"encoding/hex"
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
		if txs, ready := pool.usersTxs.SaveTx(tx, userNonce); ready {
			pool.list.Put(txs...)
		}
	}
}

func (pool *TxPool) GetTx(hash []byte) *userevent.Transaction {
	return pool.all.Get(hex.EncodeToString(hash))
}

func (pool *TxPool) Pop(size int) []*userevent.Transaction {
	txs := pool.list.Pop(size)
	return txs
}

func (pool *TxPool) Notify(txs []userevent.Transaction) {
	for _, tx := range txs {
		pool.all.Delete(tx.TransactionId())
		pool.usersTxs.Notify(tx)
		pool.list.Notify(tx)
	}
}

func (pool *TxPool) GetUserTxs(address string) *UserTxs {
	return pool.usersTxs.m[address]
}
