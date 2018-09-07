package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/EducationEKT/EKT/MPTPlus"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/round"
)

var currentBlock *Block = nil

type Block struct {
	Height       int64          `json:"height"`
	Timestamp    int64          `json:"timestamp"`
	Nonce        int64          `json:"nonce"`
	Fee          int64          `json:"fee"`
	TotalFee     int64          `json:"totalFee"`
	PreviousHash types.HexBytes `json:"previousHash"`
	CurrentHash  types.HexBytes `json:"currentHash"`
	Signature    types.HexBytes `json:"signature"`
	BlockBody    *BlockBody     `json:"-"`
	Body         types.HexBytes `json:"body"`
	Round        *round.Round   `json:"round"`
	Locker       sync.RWMutex   `json:"-"`
	StatTree     *MPTPlus.MTP   `json:"-"`
	StatRoot     types.HexBytes `json:"statRoot"`
	TxTree       *MPTPlus.MTP   `json:"-"`
	TxRoot       types.HexBytes `json:"txRoot"`
	TokenTree    *MPTPlus.MTP   `json:"-"`
	TokenRoot    types.HexBytes `json:"tokenRoot"`
}

func (block Block) GetRound() *round.Round {
	return block.Round.Clone()
}

func (block *Block) Bytes() []byte {
	block.UpdateMPTPlusRoot()
	data, _ := json.Marshal(block)
	return data
}

func (block *Block) Data() []byte {
	round := ""
	if block.Height > 0 {
		round = block.GetRound().String()
	}
	return []byte(fmt.Sprintf(
		`{"height": %d, "timestamp": %d, "nonce": %d, "fee": %d, "totalFee": %d, "previousHash": "%s", "body": "%s", "round": %s, "statRoot": "%s", "txRoot": "%s", "eventRoot": "%s", "tokenRoot": "%s"}`,
		block.Height, block.Timestamp, block.Nonce, block.Fee, block.TotalFee, hex.EncodeToString(block.PreviousHash), hex.EncodeToString(block.Body),
		round, hex.EncodeToString(block.StatRoot), hex.EncodeToString(block.TxRoot), hex.EncodeToString(block.TokenRoot),
	))
}

func (block *Block) Hash() []byte {
	return block.CurrentHash
}

func (block *Block) CaculateHash() []byte {
	block.CurrentHash = crypto.Sha3_256(block.Data())
	return block.CurrentHash
}

func (block Block) ValidateHash() bool {
	return bytes.EqualFold(block.CurrentHash, block.CaculateHash())
}

func (block *Block) NewNonce() {
	block.Nonce++
}

func (block Block) GetAccount(address []byte) (*types.Account, error) {
	value, err := block.StatTree.GetValue(address)
	if err != nil {
		return nil, err
	}
	var account types.Account
	err = json.Unmarshal(value, &account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (block Block) ExistAddress(address []byte) bool {
	return block.StatTree.ContainsKey(address)
}

func GenesisBlock(accounts []types.Account) *Block {
	block := &Block{
		Height:       0,
		Nonce:        0,
		Fee:          BackboneChainFee,
		TotalFee:     0,
		PreviousHash: nil,
		CurrentHash:  nil,
		BlockBody:    NewBlockBody(),
		Body:         nil,
		Timestamp:    0,
		Locker:       sync.RWMutex{},
		StatTree:     MPTPlus.NewMTP(db.GetDBInst()),
		StatRoot:     nil,
		TxTree:       MPTPlus.NewMTP(db.GetDBInst()),
		TxRoot:       nil,
		TokenTree:    MPTPlus.NewMTP(db.GetDBInst()),
		TokenRoot:    nil,
	}

	for _, account := range accounts {
		block.CreateGenesisAccount(account)
	}

	block.UpdateMPTPlusRoot()
	block.CaculateHash()

	return block
}

func (block *Block) CreateGenesisAccount(account types.Account) bool {
	err := block.StatTree.MustInsert(account.Address, account.ToBytes())
	if err != nil {
		return false
	}
	block.UpdateMPTPlusRoot()
	return true
}

func (block *Block) NewTransaction(tx userevent.Transaction, fee int64) *userevent.UserEventResult {
	account, err := block.GetAccount(tx.GetFrom())
	if err != nil {
		return userevent.NewUserEventResult(&tx, fee, false, "invalid from")
	}
	var receiverAccount *types.Account
	if block.ExistAddress(tx.GetTo()) {
		receiverAccount, _ = block.GetAccount(tx.GetTo())
	} else {
		receiverAccount = types.NewAccount(tx.GetTo())
	}
	var txResult *userevent.UserEventResult

	// 如果fee太少，默认使用系统最少值
	if fee < block.Fee {
		fee = block.Fee
	}

	// set fee = 0, 在官方托管节点期间，免交易费
	fee = 0

	if tx.Nonce != account.Nonce+1 {
		txResult = userevent.NewUserEventResult(&tx, fee, false, "invalid nonce")
	} else if tx.TokenAddress == "" {
		if account.GetAmount() < tx.Amount+fee {
			txResult = userevent.NewUserEventResult(&tx, fee, false, "no enough gas")
		} else {
			account.ReduceAmount(tx.Amount + fee)
			receiverAccount.AddAmount(tx.Amount)
			block.StatTree.MustInsert(tx.GetFrom(), account.ToBytes())
			block.StatTree.MustInsert(tx.GetTo(), receiverAccount.ToBytes())
			txResult = userevent.NewUserEventResult(&tx, fee, true, "")
		}
	} else {
		if account.Balances[tx.TokenAddress] < tx.Amount {
			txResult = userevent.NewUserEventResult(&tx, fee, false, "no enough amount")
		} else if account.GetAmount() < fee {
			txResult = userevent.NewUserEventResult(&tx, fee, false, "no enough gas")
		} else {
			account.Balances[tx.TokenAddress] -= tx.Amount
			account.ReduceAmount(fee)
			if receiverAccount.Balances == nil {
				receiverAccount.Balances = make(map[string]int64)
				receiverAccount.Balances[tx.TokenAddress] = 0
			}
			receiverAccount.Balances[tx.TokenAddress] += tx.Amount
			block.StatTree.MustInsert(tx.GetFrom(), account.ToBytes())
			block.StatTree.MustInsert(tx.GetTo(), receiverAccount.ToBytes())
			txResult = userevent.NewUserEventResult(&tx, fee, true, "")
		}
	}
	txId, _ := hex.DecodeString(tx.TransactionId())

	block.TxTree.MustInsert(txId, txResult.ToBytes())

	if txResult.Success {
		block.TotalFee += txResult.Fee
	}

	return txResult
}

func (block *Block) UpdateMPTPlusRoot() {
	if block.StatTree != nil {
		block.StatTree.Lock.RLock()
		block.StatRoot = block.StatTree.Root
		block.StatTree.Lock.RUnlock()
	}
	if block.TxTree != nil {
		block.TxTree.Lock.RLock()
		block.TxRoot = block.TxTree.Root
		block.TxTree.Lock.RUnlock()
	}
	if block.TokenTree != nil {
		block.TokenTree.Lock.RLock()
		block.TokenRoot = block.TokenTree.Root
		block.TokenTree.Lock.RUnlock()
	}
}

func (block *Block) RecoverMPT() {
	if block.StatTree == nil {
		block.StatTree = MPTPlus.MTP_Tree(db.GetDBInst(), block.StatRoot)
	}
	if block.TxTree == nil {
		block.TxTree = MPTPlus.MTP_Tree(db.GetDBInst(), block.TxRoot)
	}
	if block.TokenTree == nil {
		block.TokenTree = MPTPlus.MTP_Tree(db.GetDBInst(), block.TokenRoot)
	}
}

func FromBytes2Block(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil, err
	}
	block.StatTree = MPTPlus.MTP_Tree(db.GetDBInst(), block.StatRoot)
	block.TxTree = MPTPlus.MTP_Tree(db.GetDBInst(), block.TxRoot)
	block.Locker = sync.RWMutex{}
	return &block, nil
}

func NewBlock(last Block) *Block {
	block := &Block{
		Height:       last.Height + 1,
		Nonce:        0,
		Fee:          last.Fee,
		TotalFee:     0,
		PreviousHash: last.Hash(),
		Timestamp:    time.Now().UnixNano() / 1e6,
		CurrentHash:  nil,
		BlockBody:    NewBlockBody(),
		Body:         nil,
		Locker:       sync.RWMutex{},
		StatTree:     MPTPlus.MTP_Tree(db.GetDBInst(), last.StatRoot),
		TxTree:       MPTPlus.NewMTP(db.GetDBInst()),
		TokenTree:    MPTPlus.MTP_Tree(db.GetDBInst(), last.TokenRoot),
	}
	return block
}

func (block Block) ValidateNextBlock(next Block, events []userevent.IUserEvent) bool {
	// 如果不是当前的块的下一个区块，则返回false
	if !bytes.Equal(next.PreviousHash, block.Hash()) || block.Height+1 != next.Height {
		return false
	}
	return block.ValidateBlockStat(next, events)
}

func (block Block) ValidateBlockStat(next Block, events []userevent.IUserEvent) bool {
	log.Info("Validating block stat merkler proof.")

	//根据上一个区块头生成一个新的区块
	_next := NewBlock(block)

	//让新生成的区块执行peer传过来的body中的user events进行计算
	if len(events) > 0 {
		for _, event := range events {
			switch event.Type() {
			case userevent.TYPE_USEREVENT_TRANSACTION:
				tx, ok := event.(*userevent.Transaction)
				if ok {
					_next.NewTransaction(*tx, tx.Fee)
				}
			case userevent.TYPE_USEREVENT_PUBLIC_TOKEN:
				issueToken, ok := event.(*userevent.TokenIssue)
				if ok {
					_next.IssueToken(*issueToken)
				}
			}
		}
	}

	address, err := hex.DecodeString(next.Round.Peers[next.Round.CurrentIndex].Account)
	if err != nil {
		log.Crit("invalid address")
		return false
	}
	_next.UpdateMiner(address)

	// 更新默克尔树根
	_next.UpdateMPTPlusRoot()

	// 判断默克尔根是否相同
	if !bytes.Equal(next.TxRoot, _next.TxRoot) ||
		!bytes.Equal(next.StatRoot, _next.StatRoot) ||
		!bytes.Equal(next.TokenRoot, _next.TokenRoot) {
		return false
	}

	return true
}

func (block *Block) UpdateMiner(address []byte) {
	account, err := block.GetAccount(address)
	if account == nil || err != nil {
		account = types.NewAccount(address)
	}
	account.Amount += block.TotalFee

	err = block.StatTree.MustInsert(address, account.ToBytes())
	if err != nil {
		log.Crit("Update miner failed, %s", err.Error())
	}
}

func (block *Block) Sign(privKey []byte) error {
	Signature, err := crypto.Crypto(crypto.Sha3_256(block.CurrentHash), privKey)
	block.Signature = Signature
	return err
}

func (block *Block) IssueToken(event userevent.TokenIssue) *userevent.UserEventResult {
	eventId, _ := hex.DecodeString(event.EventId())
	account, err := block.GetAccount(event.GetFrom())

	if err != nil {
		eventResult := userevent.NewUserEventResult(&event, 0, false, err.Error())
		block.TxTree.MustInsert(eventId, eventResult.ToBytes())
		return eventResult
	}

	if account.GetNonce()+1 != event.GetNonce() {
		eventResult := userevent.NewUserEventResult(&event, 0, false, "Invalid nonce")
		block.TxTree.MustInsert(eventId, eventResult.ToBytes())
		return eventResult
	}

	fee := int64(500 * 1e8)
	if account.Amount < fee {
		eventResult := userevent.NewUserEventResult(&event, 0, false, "no enough fee")
		block.TxTree.MustInsert(eventId, eventResult.ToBytes())
		return eventResult
	}

	account.Amount -= fee
	if len(account.Balances) == 0 {
		account.Balances = make(map[string]int64)
	}
	total := Decimals(event.Token.Decimals) * event.Token.Total
	account.Balances[hex.EncodeToString(event.Token.Address())] = total
	account.Nonce++

	block.TotalFee += fee

	block.TokenTree.MustInsert(event.Token.Address(), event.Token.Bytes())
	block.StatTree.MustInsert(event.GetFrom(), account.ToBytes())
	eventResult := userevent.NewUserEventResult(&event, fee, true, "")
	block.TxTree.MustInsert(eventId, eventResult.ToBytes())

	return eventResult
}

func Decimals(decimal int64) int64 {
	result := int64(1)
	for i := int64(0); i < decimal; i++ {
		result *= 10
	}
	return result
}
