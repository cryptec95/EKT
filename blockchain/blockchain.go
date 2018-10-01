package blockchain

import (
	"bytes"
	"fmt"
	"github.com/EducationEKT/EKT/ctxlog"
	"sync"
	"time"

	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/pool"
)

const (
	BackboneBlockInterval = 3 * time.Second
)

type BlockChain struct {
	ChainId       int64
	currentLocker sync.RWMutex
	header        Header
	currentHeight int64
	Locker        sync.RWMutex
	Pool          *pool.TxPool
	PackLock      sync.RWMutex
}

func NewBlockChain(chainId int64) *BlockChain {
	return &BlockChain{
		ChainId:       chainId,
		Locker:        sync.RWMutex{},
		currentLocker: sync.RWMutex{},
		Pool:          pool.NewTxPool(),
		PackLock:      sync.RWMutex{},
	}
}

func (chain *BlockChain) LastHeader() Header {
	chain.currentLocker.RLock()
	defer chain.currentLocker.RUnlock()
	return chain.header
}

func (chain *BlockChain) SetLastHeader(header Header) {
	chain.currentLocker.Lock()
	defer chain.currentLocker.Unlock()
	chain.header = header
	chain.currentHeight = header.Height
}

func (chain *BlockChain) GetLastHeight() int64 {
	chain.currentLocker.RLock()
	defer chain.currentLocker.RUnlock()
	return chain.currentHeight
}

func (chain *BlockChain) PackTime() time.Duration {
	return BackboneBlockInterval - 500*time.Millisecond
}

func (chain *BlockChain) PackTransaction(ctxlog *ctxlog.ContextLog, block *Block) {
	defer block.Finish()
	eventTimeout := time.After(chain.PackTime())

	start := time.Now().UnixNano()
	started := false
	numTx := 0
	for {
		flag := false
		select {
		case <-eventTimeout:
			flag = true
		default:
			txs := chain.Pool.Pop(20)
			if len(txs) > 0 {
				if !started {
					started = true
					start = time.Now().UnixNano()
				}
				for _, tx := range txs {
					block.NewTransaction(*tx)
				}
				numTx += len(txs)
			}
		}
		if flag {
			break
		}
	}

	end := time.Now().UnixNano()
	log.Debug("Total tx: %d, Total time: %d ns, TPS: %d. \n", numTx, end-start, numTx*1e9/int(end-start))
}

// 当区块写入区块时，notify交易池，一些nonce比较大的交易可以进行打包
func (chain *BlockChain) NotifyPool(txs []userevent.Transaction) {
	chain.Pool.Notify(txs)
}

func (chain *BlockChain) NewTransaction(tx *userevent.Transaction) bool {
	block := chain.LastHeader()
	account, err := block.GetAccount(tx.GetFrom())
	if err != nil || account.GetNonce() >= tx.GetNonce() {
		return false
	}
	chain.Pool.Park(tx, account.GetNonce())
	return true
}

func (chain *BlockChain) ValidateBlock(next Block) bool {
	newBlock := CreateBlock(chain.LastHeader(), next.Miner)
	newBlock.GetHeader().Timestamp = next.GetHeader().Timestamp
	for _, tx := range next.GetTransactions() {
		newBlock.NewTransaction(tx)
	}
	newBlock.Finish()
	if !bytes.EqualFold(newBlock.Hash, next.Hash) {
		return false
	}
	return true
}
