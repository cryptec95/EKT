package pool

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/core/userevent"
	"sync"
)

type TransactionDict struct {
	All  map[string]*userevent.Transaction `json:"all"`
	lock sync.RWMutex
}

func NewTransactionDict() *TransactionDict {
	return &TransactionDict{
		All:  make(map[string]*userevent.Transaction),
		lock: sync.RWMutex{},
	}
}

func (dict TransactionDict) Range(f func(hash string, tx *userevent.Transaction) bool) {
	dict.lock.RLock()
	defer dict.lock.RUnlock()

	for key, value := range dict.All {
		if !f(key, value) {
			break
		}
	}
}

func (dict *TransactionDict) Get(hash string) *userevent.Transaction {
	dict.lock.RLock()
	defer dict.lock.RUnlock()

	return dict.All[hash]
}

func (dict *TransactionDict) Save(tx *userevent.Transaction) {
	dict.lock.Lock()
	defer dict.lock.Unlock()

	dict.All[hex.EncodeToString(tx.TxId())] = tx
}

func (dict *TransactionDict) Delete(hash string) {
	dict.lock.Lock()
	defer dict.lock.Unlock()

	delete(dict.All, hash)
}
