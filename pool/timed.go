package pool

import (
	"bytes"
	"github.com/EducationEKT/EKT/core/userevent"
	"sync"
)

type TxTimedList struct {
	list   []*userevent.Transaction
	locker sync.RWMutex
}

func NewTimedList() *TxTimedList {
	return &TxTimedList{
		list:   make([]*userevent.Transaction, 0),
		locker: sync.RWMutex{},
	}
}

func (list TxTimedList) Put(tx *userevent.Transaction) {
	list.locker.Lock()
	list.list = append(list.list, tx)
	list.locker.Unlock()
}

func (list TxTimedList) Pop(size int) []*userevent.Transaction {
	list.locker.Lock()
	defer list.locker.Unlock()
	if len(list.list) < size {
		result := list.list
		list.list = list.list[:0]
		return result
	} else {
		result := list.list[:size]
		list.list = list.list[size:]
		return result
	}
}

func (list TxTimedList) Notify(tx userevent.Transaction) {
	list.locker.Lock()
	defer list.locker.Unlock()
	for i, _tx := range list.list {
		if bytes.EqualFold(tx.TxId(), _tx.TxId()) {
			list.list = append(list.list[:i], list.list[i+1:]...)
			break
		}
	}
}
