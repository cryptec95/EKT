package blockchain

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"errors"

	"encoding/hex"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/i_consensus"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/param"
	"github.com/EducationEKT/EKT/pool"
	"github.com/EducationEKT/EKT/round"
	"os"
)

var BackboneChainId int64 = 1

const (
	BackboneConsensus     = i_consensus.DBFT
	BackboneBlockInterval = 3 * time.Second
	BackboneChainFee      = 510000
)

const (
	InitStatus      = 0
	StartPackStatus = 100
)

type BlockChain struct {
	ChainId       int64
	Consensus     i_consensus.ConsensusType
	currentLocker sync.RWMutex
	currentBlock  Block
	currentHeight int64
	Locker        sync.RWMutex
	Status        int
	Fee           int64
	Difficulty    []byte
	Pool          *pool.TxPool
	BlockInterval time.Duration
	PackLock      sync.RWMutex
}

func NewBlockChain(chainId int64, consensusType i_consensus.ConsensusType, fee int64, difficulty []byte, interval time.Duration) *BlockChain {
	return &BlockChain{
		ChainId:       chainId,
		Consensus:     consensusType,
		Locker:        sync.RWMutex{},
		currentLocker: sync.RWMutex{},
		Status:        InitStatus, // 100 正在计算MTProot, 150停止计算root,开始计算block Hash
		Fee:           fee,
		Difficulty:    difficulty,
		Pool:          pool.NewTxPool(),
		currentHeight: 0,
		BlockInterval: interval,
		PackLock:      sync.RWMutex{},
	}
}

func (chain *BlockChain) GetLastBlock() Block {
	chain.currentLocker.RLock()
	defer chain.currentLocker.RUnlock()
	return chain.currentBlock
}

func (chain *BlockChain) SetLastBlock(block *Block) {
	chain.currentLocker.Lock()
	defer chain.currentLocker.Unlock()
	block.RecoverMPT()
	chain.currentBlock = *block
	chain.currentHeight = block.Height
}

func (chain *BlockChain) GetLastHeight() int64 {
	chain.currentLocker.RLock()
	defer chain.currentLocker.RUnlock()
	return chain.currentHeight
}

func (chain *BlockChain) PackSignal(ctxLog *ctxlog.ContextLog, height int64) *Block {
	chain.PackLock.Lock()
	defer chain.PackLock.Unlock()
	if chain.Status != StartPackStatus {
		defer func() {
			if r := recover(); r != nil {
				log.Crit("Panic while pack. %v", r)
			}
			chain.Status = InitStatus
		}()
		log.Debug("Start pack block at height %d .\n", chain.GetLastHeight()+1)

		block := chain.WaitAndPack(ctxLog)

		return block
	}
	return nil
}

func (chain *BlockChain) GetBlockByHeight(height int64) (*Block, error) {
	if height > chain.GetLastHeight() {
		return nil, errors.New("Invalid height")
	}
	key := chain.GetBlockByHeightKey(height)
	data, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("Too heigher.")
	}
	block, err := FromBytes2Block(data)
	if block.Height != height {
		return nil, errors.New("Can not get block from db.")
	}
	return block, err
}

func (chain *BlockChain) GetBlockByHeightKey(height int64) []byte {
	return []byte(fmt.Sprint(`GetBlockByHeight: _%d_%d`, chain.ChainId, height))
}

func (chain *BlockChain) SaveBlock(block Block) {
	chain.Locker.Lock()
	defer chain.Locker.Unlock()

	if chain.GetLastHeight()+1 == block.Height || block.Height == 0 {
		log.Info("Saving block %s to database.", string(block.Bytes()))
		db.GetDBInst().Set(block.Hash(), block.Data())
		db.GetDBInst().Set(chain.GetBlockByHeightKey(block.Height), block.Bytes())
		db.GetDBInst().Set(chain.CurrentBlockKey(), block.Bytes())
		chain.SetLastBlock(&block)
	}
}

func (chain *BlockChain) LastBlock() (*Block, error) {
	var err error = nil
	var block *Block
	if currentBlock == nil {
		key := chain.CurrentBlockKey()
		data, err := db.GetDBInst().Get(key)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &block)
		if err != nil {
			return nil, err
		}
		currentBlock = block
		return block, err
	}
	return currentBlock, err
}

func (chain *BlockChain) CurrentBlockKey() []byte {
	return []byte(fmt.Sprintf("CurrentBlockKey_%d", chain.ChainId))
}

func (chain *BlockChain) PackTime() time.Duration {
	lastBlock := chain.GetLastBlock()
	d := time.Now().UnixNano() - lastBlock.Timestamp*1e6
	return time.Duration(int64(chain.BlockInterval)-d) / 2
}

func (chain *BlockChain) WaitAndPack(ctxLog *ctxlog.ContextLog) *Block {
	eventTimeout := time.After(chain.PackTime())
	block := NewBlock(chain.GetLastBlock())
	if block.Fee <= 0 {
		block.Fee = chain.Fee
	}

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
							block.NewTransaction(*tx, tx.Fee)
						}
					case userevent.TYPE_USEREVENT_PUBLIC_TOKEN:
						issueToken, ok := event.(*userevent.TokenIssue)
						if ok {
							numTx++
							block.IssueToken(*issueToken)
						}
					}

					block.BlockBody.AddEvent(event)
				}
			}
		}
		if flag {
			break
		}
	}

	address, err := hex.DecodeString(conf.EKTConfig.Node.Account)
	if err != nil {
		log.Crit("Invalid config")
		os.Exit(-1)
	}
	block.UpdateMiner(address)

	end := time.Now().UnixNano()
	log.Debug("Total tx: %d, Total time: %d ns, TPS: %d. \n", numTx, end-start, numTx*1e9/int(end-start))

	chain.UpdateBody(block)
	block.UpdateMPTPlusRoot()

	if block.Round == nil {
		block.Round = &round.Round{
			Peers:        param.MainChainDelegateNode,
			CurrentIndex: 0,
		}
	} else {
		// 判断是否需要进入下一个round
		block.Round = chain.GetLastBlock().Round
		block.Round.CurrentIndex = block.Round.MyIndex()
	}

	return block
}

func (chain *BlockChain) UpdateBody(block *Block) {
	bodyData := block.BlockBody.Bytes()
	block.Body = crypto.Sha3_256(bodyData)
	db.GetDBInst().Set(block.Body, bodyData)
}

// 当区块写入区块时，notify交易池，一些nonce比较大的交易可以进行打包
func (chain *BlockChain) NotifyPool(block Block) {
	if block.BlockBody == nil {
		return
	}

	chain.Pool.Notify <- block.BlockBody.Events
}

func (chain *BlockChain) NewUserEvent(event userevent.IUserEvent) bool {
	block := chain.GetLastBlock()
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
	block := chain.GetLastBlock()
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

func (chain *BlockChain) ValidateNextBlock(ctxlog *ctxlog.ContextLog, block Block, events []userevent.IUserEvent) bool {
	if !chain.GetLastBlock().ValidateNextBlock(block, events) {
		ctxlog.Log("Validate", false)
		return false
	}
	ctxlog.Log("Validate", true)
	return true
}
