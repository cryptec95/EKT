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
	Index  int								`json:"index"`    //nonce在nonces列表中的腳標位置
}

func NewUserTxs(nonce int64) *UserTxs {
	return &UserTxs{
		Txs:    make(map[int64]*userevent.Transaction),
		Nonces: NewNonceList(),
		Nonce:  nonce,
		Index:  0,
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
			//sorted.Nonce = lastNonce
			//sorted.clearNonces()
			sorted.Notify(lastNonce)
			return list, true
		}
	}
	return nil, false
}

func (sorted *UserTxs) clearNonces(){

	for i := 0; i < len(*sorted.Nonces); i++ {
		nonce := (*sorted.Nonces)[i]
		if nonce == sorted.Nonce {
			sorted.Index = i
		}
	}
}

func (sorted *UserTxs) Notify(nonce int64) {
	sorted.Nonce = nonce
	for i := 0; i < len(*sorted.Nonces); i++ {
		nonce := (*sorted.Nonces)[i]
		if nonce == sorted.Nonce {
			sorted.Index = i
		}
	}

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
	m.Nonce = nonce
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
		//userTxs.Nonce = tx.Nonce
		//userTxs.clearNonces()
	}
	return nil, false
}
