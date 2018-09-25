package blockchain

import (
	"encoding/hex"
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
	return BackboneBlockInterval
}

func (chain *BlockChain) PackTransaction(block *Block) {
	defer block.Finish()

	header := block.GetHeader()
	eventTimeout := time.After(chain.PackTime())

	start := time.Now().UnixNano()
	started := false
	numTx := 0
	for {
		flag := false
		select {
		case <-eventTimeout:
			flag = true
			break
		default:
			multiFetcher := pool.NewMultiFetcher(10)
			chain.Pool.MultiFetcher <- multiFetcher
			events := <-multiFetcher.Chan
			if len(events) > 0 {
				if !started {
					started = true
					start = time.Now().UnixNano()
				}
				for _, event := range events {
					switch event.Type() {
					case userevent.TYPE_USEREVENT_TRANSACTION:
						tx, ok := event.(*userevent.Transaction)
						if ok {
							numTx++
							header.NewTransaction(*tx)
						}
					}
				}
			}
		}
		if flag {
			break
		}
	}

	header.UpdateMiner()

	end := time.Now().UnixNano()
	log.Debug("Total tx: %d, Total time: %d ns, TPS: %d. \n", numTx, end-start, numTx*1e9/int(end-start))
}

// 当区块写入区块时，notify交易池，一些nonce比较大的交易可以进行打包
func (chain *BlockChain) NotifyPool(transactions []userevent.Transaction) {
	txIds := make([]string, 0)
	for _, tx := range transactions {
		txIds = append(txIds, hex.EncodeToString(tx.TxId()))
	}
	chain.Pool.Notify <- txIds
}

func (chain *BlockChain) NewUserEvent(event userevent.IUserEvent) bool {
	block := chain.LastHeader()
	account, err := block.GetAccount(event.GetFrom())
	if err != nil {
		return false
	}
	if account.GetNonce() >= event.GetNonce() {
		return false
	}
	if account.GetNonce()+1 == event.GetNonce() {
		chain.Pool.SingleReady <- event
	} else {
		chain.Pool.SingleBlock <- event
	}
	return true
}

func (chain *BlockChain) NewTransaction(tx *userevent.Transaction) bool {
	block := chain.LastHeader()
	account, err := block.GetAccount(tx.GetFrom())
	if err != nil {
		return false
	}
	if account.GetNonce() >= tx.GetNonce() {
		return false
	}
	if account.GetNonce()+1 == tx.GetNonce() {
		chain.Pool.SingleReady <- tx
	} else {
		chain.Pool.SingleBlock <- tx
	}
	return true
}
