package blockchain

import (
	"sync"
	"time"

	"encoding/hex"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/pool"
)

var BackboneChainId int64 = 1

const (
	BackboneBlockInterval = 3 * time.Second
	BackboneChainFee      = 510000
	InitStatus            = 0
	StartPackStatus       = 100
)

type BlockChain struct {
	ChainId       int64
	currentLocker sync.RWMutex
	currentBlock  Header
	currentHeight int64
	Locker        sync.RWMutex
	Status        int
	Pool          *pool.TxPool
	PackLock      sync.RWMutex
}

func NewBlockChain() *BlockChain {
	return &BlockChain{
		Locker:        sync.RWMutex{},
		currentLocker: sync.RWMutex{},
		Status:        InitStatus, // 100 正在计算MTProot, 150停止计算root,开始计算block Hash
		Pool:          pool.NewTxPool(),
		PackLock:      sync.RWMutex{},
	}
}

func (chain *BlockChain) LastBlock() Header {
	chain.currentLocker.RLock()
	defer chain.currentLocker.RUnlock()
	return chain.currentBlock
}

func (chain *BlockChain) SetLastBlock(block Header) {
	chain.currentLocker.Lock()
	defer chain.currentLocker.Unlock()
	chain.currentBlock = block
	chain.currentHeight = block.Height
}

func (chain *BlockChain) GetLastHeight() int64 {
	chain.currentLocker.RLock()
	defer chain.currentLocker.RUnlock()
	return chain.currentHeight
}

func (chain *BlockChain) PackSignal(ctxLog *ctxlog.ContextLog, height int64) *Header {
	chain.PackLock.Lock()
	defer chain.PackLock.Unlock()
	if chain.Status != StartPackStatus {
		defer func() {
			if r := recover(); r != nil {
				log.PrintStack("blockchain.PackSignal")
			}
			chain.Status = InitStatus
		}()
		log.Debug("StartNode pack block at height %d .\n", chain.GetLastHeight()+1)

		block := chain.WaitAndPack(ctxLog)

		return block
	}
	return nil
}

func (chain *BlockChain) PackTime() time.Duration {
	lastBlock := chain.LastBlock()
	d := time.Now().UnixNano() - lastBlock.Timestamp*1e6
	return time.Duration(int64(BackboneBlockInterval)-d) / 2
}

func (chain *BlockChain) WaitAndPack(ctxLog *ctxlog.ContextLog) *Header {
	eventTimeout := time.After(chain.PackTime())
	lastBlock := chain.LastBlock()
	coinbase, _ := hex.DecodeString(conf.EKTConfig.Node.Account)
	block := NewHeader(lastBlock, lastBlock.CaculateHash(), coinbase)

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
							block.NewTransaction(*tx)
						}
					case userevent.TYPE_USEREVENT_PUBLIC_TOKEN:
						issueToken, ok := event.(*userevent.TokenIssue)
						if ok {
							numTx++
							block.IssueToken(*issueToken)
						}
					}
				}
			}
		}
		if flag {
			break
		}
	}

	block.UpdateMiner()

	end := time.Now().UnixNano()
	log.Debug("Total tx: %d, Total time: %d ns, TPS: %d. \n", numTx, end-start, numTx*1e9/int(end-start))

	return block
}

// 当区块写入区块时，notify交易池，一些nonce比较大的交易可以进行打包
func (chain *BlockChain) NotifyPool(block Header) {
	//if block.BlockBody == nil {
	//	return
	//}

	//chain.Pool.Notify <- block.BlockBody.Events
}

func (chain *BlockChain) NewUserEvent(event userevent.IUserEvent) bool {
	block := chain.LastBlock()
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
	block := chain.LastBlock()
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
