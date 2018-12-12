package blockchain

import (
	"time"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/pool"
	"github.com/EducationEKT/EKT/schema"
)

const (
	BackboneBlockInterval = 3 * time.Second
)

type BlockChain struct {
	ChainId       int64
	header        Header
	currentHeight int64
	Pool          *pool.TxPool
}

func NewBlockChain(chainId int64) *BlockChain {
	return &BlockChain{
		ChainId: chainId,
		Pool:    pool.NewTxPool(),
	}
}

func (chain *BlockChain) LastHeader() Header {
	return chain.header
}

func (chain *BlockChain) SetLastHeader(header Header) {
	chain.header = header
	chain.currentHeight = header.Height
}

func (chain *BlockChain) GetLastHeight() int64 {
	return chain.currentHeight
}

func (chain *BlockChain) PackTime(block *Block) time.Duration {
	return time.Duration(block.GetHeader().Timestamp+2000-time.Now().UnixNano()/1e6) * 1e6
}

func (chain *BlockChain) PackTransaction(clog *ctxlog.ContextLog, block *Block) {
	defer block.Finish()
	t := chain.PackTime(block)
	eventTimeout := time.After(t)

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
					receipt := block.NewTransaction(*tx)
					log.LogErr(block.Header.TxRoot.MustInsert(tx.TxId(), tx.Bytes()))
					log.LogErr(block.Header.ReceiptRoot.MustInsert(tx.TxId(), receipt.Bytes()))
					receiptDetail := userevent.ReceiptDetail{
						Receipt:     *receipt,
						BlockNumber: block.GetHeader().Height,
						Index:       int64(numTx),
					}
					log.LogErr(db.GetDBInst().Set(schema.GetReceiptByTxHashKey(chain.ChainId, tx.TransactionId()), receiptDetail.Bytes()))
					block.Transactions = append(block.Transactions, *tx)
					block.TransactionReceipts = append(block.TransactionReceipts, *receipt)
				}
				numTx += len(txs)
			} else {
				chain.Pool.Promote(*block.GetHeader().StatTree)
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
	if err != nil || account == nil {
		if tx.GetNonce() != 1 {
			return false
		}
		account = types.NewAccount(tx.From)
	}
	if account.GetNonce() >= tx.GetNonce() {
		return false
	}
	chain.Pool.Park(tx, account.GetNonce())
	return true
}

func (chain *BlockChain) ValidateBlock(next Block) bool {
	lastHeader := chain.LastHeader()
	newBlock := CreateBlock(lastHeader, next.GetHeader().Timestamp, next.Miner)
	receipts := next.GetTxReceipts()
	for i, tx := range next.GetTransactions() {
		if chain.Pool.GetTx(tx.TxId()) == nil {
			if !userevent.ValidateTransaction(tx) {
				return false
			} else {
				logErr(db.GetDBInst().Set(tx.TxId(), tx.Bytes()))
			}
		}
		log.LogErr(newBlock.GetHeader().TxRoot.MustInsert(tx.TxId(), tx.Bytes()))
		log.LogErr(newBlock.GetHeader().ReceiptRoot.MustInsert(tx.TxId(), tx.Bytes()))
		var receipt *userevent.TransactionReceipt
		if len(tx.To) == types.ContractAddressLength {
			_receipt := receipts[i]
			if !_receipt.Success {
				newBlock.Transactions = append(newBlock.Transactions, tx)
				newBlock.TransactionReceipts = append(newBlock.TransactionReceipts, _receipt)
				continue
			}
		}
		receipt = newBlock.NewTransaction(tx)
		newBlock.Transactions = append(newBlock.Transactions, tx)
		newBlock.TransactionReceipts = append(newBlock.TransactionReceipts, *receipt)
	}
	newBlock.Finish()
	return newBlock.GetHeader().Equal(*next.GetHeader())
}
