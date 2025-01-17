package pool

import (
	"encoding/hex"
	"sync"

	"github.com/EducationEKT/EKT/core/userevent"
)

type NonceList []int64

func NewNonceList() *NonceList {
	list := make(NonceList, 0)
	return &list
}

func (list *NonceList) Insert(nonce int64) {
	if len(*list) == 0 {
		*list = append(*list, nonce)
		return
	}
	if nonce > (*list)[len(*list)-1] {
		*list = append(*list, nonce)
		return
	}
	newList := make([]int64, 0)
	for i, n := range *list {
		if n < nonce {
			newList = append(newList, n)
		} else {
			newList = append(newList, nonce)
			newList = append(newList, (*list)[i:]...)
			*list = newList
			return
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
	Index  int                              `json:"index"`
}

func NewUserTxs(nonce int64) *UserTxs {
	return &UserTxs{
		Txs:    make(map[int64]*userevent.Transaction),
		Nonces: NewNonceList(),
		Nonce:  nonce,
		Index:  -1,
	}
}

func (sorted *UserTxs) Save(tx *userevent.Transaction) ([]*userevent.Transaction, bool) {
	if sorted.Txs[tx.Nonce] != nil {
		return nil, false
	} else {
		sorted.Nonces.Insert(tx.Nonce)
		sorted.Txs[tx.Nonce] = tx

		if tx.Nonce == sorted.Nonce+1 {
			list := make([]*userevent.Transaction, 0)
			list = append(list, tx)
			lastNonce := tx.Nonce
			for i := 0; i < len(*sorted.Nonces); i++ {
				if (*sorted.Nonces)[i] == lastNonce+1 {
					lastNonce++
					_tx := sorted.Txs[lastNonce]
					list = append(list, _tx)
				}
			}
			sorted.Nonce = lastNonce
			sorted.updateIndex()
			return list, true
		}
	}
	return nil, false
}

func (sorted *UserTxs) updateIndex() {
	for i := 0; i < len(*sorted.Nonces); i++ {
		nonce := (*sorted.Nonces)[i]
		if nonce == sorted.Nonce {
			sorted.Index = i
			return
		}
	}
	sorted.Index = -1
}

func (sorted *UserTxs) clearNonces() {
	newNonces := NewNonceList()
	for i := 0; i < len(*sorted.Nonces); i++ {
		nonce := (*sorted.Nonces)[i]
		if nonce > sorted.Nonce {
			newNonces.Insert(nonce)
		} else {
			delete(sorted.Txs, nonce)
		}
	}
	sorted.Nonces = newNonces
}

func (sorted *UserTxs) Notify(nonce int64) {
	sorted.Nonce = nonce
	newNonces := NewNonceList()
	for i := 0; i < len(*sorted.Nonces); i++ {
		nonce := (*sorted.Nonces)[i]
		if nonce > sorted.Nonce {
			newNonces.Insert(nonce)
		} else {
			delete(sorted.Txs, nonce)
		}
	}
	sorted.Nonces = newNonces
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
	M      map[string]*UserTxs `json:"m"`
	locker sync.RWMutex
}

func NewUsersTxs() *UsersTxs {
	return &UsersTxs{
		M:      make(map[string]*UserTxs),
		locker: sync.RWMutex{},
	}
}

func (m *UserTxs) Promote(address []byte, nonce int64) ([]*userevent.Transaction, bool) {
	list := make([]*userevent.Transaction, 0)
	lastNonce := nonce
	for i := 0; i < len(*m.Nonces); i++ {
		if (*m.Nonces)[i] == lastNonce+1 {
			lastNonce++
			_tx := m.Txs[lastNonce]
			list = append(list, _tx)
		}
	}
	m.Nonce = lastNonce
	if lastNonce > nonce {
		return list, true
	}
	return nil, false
}

func (m *UsersTxs) SaveTx(tx *userevent.Transaction, userNonce int64) ([]*userevent.Transaction, bool) {
	m.locker.Lock()
	defer m.locker.Unlock()
	userTxs := m.M[hex.EncodeToString(tx.From)]
	if userTxs == nil {
		userTxs = NewUserTxs(userNonce)
	}
	if userTxs.Nonce >= tx.Nonce {
		return nil, false
	}
	txs, ready := userTxs.Save(tx)
	m.M[hex.EncodeToString(tx.From)] = userTxs
	return txs, ready
}

func (m *UsersTxs) Notify(tx userevent.Transaction) ([]*userevent.Transaction, bool) {
	m.locker.Lock()
	defer m.locker.Unlock()
	userTxs := m.M[hex.EncodeToString(tx.From)]
	if userTxs != nil {
		userTxs.Notify(tx.Nonce)
	}
	return nil, false
}
