package pool

import (
	"bytes"
	"sync"

	"github.com/EducationEKT/EKT/core/userevent"
)

type TxTimedList struct {
	List   []*userevent.Transaction `json:"List"`
	M      map[string]bool          `json:"M"`
	locker sync.RWMutex
}

func NewTimedList() *TxTimedList {
	return &TxTimedList{
		List:   make([]*userevent.Transaction, 0),
		M:      make(map[string]bool),
		locker: sync.RWMutex{},
	}
}

func (list *TxTimedList) Put(txs ...*userevent.Transaction) {
	list.locker.Lock()
	for _, tx := range txs {
		if !list.M[tx.TransactionId()] {
			list.M[tx.TransactionId()] = true
			list.List = append(list.List, tx)
		}
	}
	list.locker.Unlock()
}

func (list *TxTimedList) Pop(size int) []*userevent.Transaction {
	list.locker.Lock()
	defer list.locker.Unlock()
	if len(list.List) < size {
		result := list.List
		list.List = list.List[:0]
		for _, tx := range result {
			delete(list.M, tx.TransactionId())
		}
		return result
	} else {
		result := list.List[:size]
		list.List = list.List[size:]
		for _, tx := range result {
			delete(list.M, tx.TransactionId())
		}
		return result
	}
}

func (list *TxTimedList) Notify(tx userevent.Transaction) {
	list.locker.Lock()
	defer list.locker.Unlock()
	if list.M[tx.TransactionId()] {
		for i, _tx := range list.List {
			if bytes.Equal(tx.TxId(), _tx.TxId()) {
				list.List = append(list.List[:i], list.List[i+1:]...)
				break
			}
		}
	}
}
