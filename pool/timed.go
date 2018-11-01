package pool

import (
	"bytes"
	"github.com/EducationEKT/EKT/core/userevent"
	"sync"
)

type TxTimedList struct {
	list   []*userevent.Transaction
	m      map[string]bool
	locker sync.RWMutex
}

func NewTimedList() *TxTimedList {
	return &TxTimedList{
		list:   make([]*userevent.Transaction, 0),
		m:      make(map[string]bool),
		locker: sync.RWMutex{},
	}
}

func (list *TxTimedList) Put(txs ...*userevent.Transaction) {
	list.locker.Lock()
	list.list = append(list.list, txs...)
	for _, tx := range txs {
		list.m[tx.TransactionId()] = true
	}
	list.locker.Unlock()
}

func (list *TxTimedList) Pop(size int) []*userevent.Transaction {
	list.locker.Lock()
	defer list.locker.Unlock()
	if len(list.list) < size {
		result := list.list
		list.list = list.list[:0]
		for _, tx := range result {
			delete(list.m, tx.TransactionId())
		}
		return result
	} else {
		result := list.list[:size]
		list.list = list.list[size:]
		for _, tx := range result {
			delete(list.m, tx.TransactionId())
		}
		return result
	}
}

func (list *TxTimedList) Notify(tx userevent.Transaction) {
	list.locker.Lock()
	defer list.locker.Unlock()
	if list.m[tx.TransactionId()] {
		for i, _tx := range list.list {
			if bytes.EqualFold(tx.TxId(), _tx.TxId()) {
				list.list = append(list.list[:i], list.list[i+1:]...)
				break
			}
		}
	}
}
