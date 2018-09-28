package pool

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/core/userevent"
	"sync"
)

type NonceList []int64

func NewNonceList() *NonceList {
	list := make(NonceList, 0)
	return &list
}

func (list *NonceList) Insert(nonce int64) {
	if len(*list) == 0 {
		*list = append(*list, nonce)
	}
	newList := make([]int64, 0)
	for i, n := range *list {
		if n < nonce {
			newList = append(newList, n)
		} else {
			newList = append(newList, nonce)
			newList = append(newList, (*list)[i:]...)
			*list = newList
		}
	}
}

func (list *NonceList) Delete(nonce int64) {
	newList := NewNonceList()
	for _, n := range *list {
		if n != nonce {
			*newList = append(*newList, n)
		}
	}
	*list = *newList
}

type UserTxs struct {
	Txs    map[int64]*userevent.Transaction `json:"txs"`
	Nonces *NonceList                       `json:"nonces"`
	Nonce  int64                            `json:"nonce"`
}

func NewUserTxs(nonce int64) *UserTxs {
	return &UserTxs{
		Txs:    make(map[int64]*userevent.Transaction),
		Nonces: NewNonceList(),
		Nonce:  nonce,
	}
}

func (sorted *UserTxs) Save(tx *userevent.Transaction) (ready bool) {
	if sorted.Txs[tx.Nonce] == nil {
		sorted.Nonces.Insert(tx.Nonce)
		sorted.Txs[tx.Nonce] = tx

		if tx.Nonce == sorted.Nonce+1 {
			sorted.Nonce++
			return true
		}
	}
	return false
}

func (sorted *UserTxs) Remove(tx userevent.Transaction) {
	if _, exist := sorted.Txs[tx.Nonce]; exist {
		delete(sorted.Txs, tx.Nonce)
	}
	if sorted.Nonce < tx.Nonce {
		sorted.Nonce = tx.Nonce
	}
	sorted.Nonces.Delete(tx.Nonce)
}

type UsersTxs struct {
	m      map[string]*UserTxs
	locker sync.RWMutex
}

func NewUsersTxs() *UsersTxs {
	return &UsersTxs{
		m:      make(map[string]*UserTxs),
		locker: sync.RWMutex{},
	}
}

func (m *UsersTxs) SaveTx(tx *userevent.Transaction, userNonce int64) (ready bool) {
	m.locker.Lock()
	defer m.locker.Unlock()
	userTxs := m.m[hex.EncodeToString(tx.From)]
	if userTxs == nil {
		userTxs = NewUserTxs(userNonce)
	}
	return userTxs.Save(tx)
}

func (m *UsersTxs) Remove(tx userevent.Transaction) {
	m.locker.Lock()
	defer m.locker.Unlock()
	userTxs := m.m[hex.EncodeToString(tx.From)]
	if userTxs != nil {
		userTxs.Remove(tx)
	}
}
