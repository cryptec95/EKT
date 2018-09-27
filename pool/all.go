package pool

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/core/userevent"
	"sync"
)

type TransactionDict struct {
	all  map[string]*userevent.Transaction
	lock sync.RWMutex
}

func NewTransactionDict() *TransactionDict {
	return &TransactionDict{
		all:  make(map[string]*userevent.Transaction),
		lock: sync.RWMutex{},
	}
}

func (dict TransactionDict) Range(f func(hash string, tx *userevent.Transaction) bool) {
	dict.lock.RLock()
	defer dict.lock.RUnlock()

	for key, value := range dict.all {
		if !f(key, value) {
			break
		}
	}
}

func (dict *TransactionDict) Get(hash string) *userevent.Transaction {
	dict.lock.RLock()
	defer dict.lock.RUnlock()

	return dict.all[hash]
}

func (dict *TransactionDict) Save(tx *userevent.Transaction) {
	dict.lock.Lock()
	defer dict.lock.Unlock()

	dict.all[hex.EncodeToString(tx.TxId())] = tx
}

func (dict *TransactionDict) Delete(hash string) {
	dict.lock.Lock()
	defer dict.lock.Unlock()

	delete(dict.all, hash)
}
