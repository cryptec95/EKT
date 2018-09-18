package consensus

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/EducationEKT/EKT/encapdb"
	"xserver/x_http/x_resp"

	"sync"
	"time"

	"errors"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/param"
	"github.com/EducationEKT/EKT/util"
	"runtime"
	"strings"
)

type DbftConsensus struct {
	Round        *types.Round
	Blockchain   *blockchain.BlockChain
	BlockManager *blockchain.BlockManager
	VoteResults  blockchain.VoteResults
	Locker       sync.RWMutex
}

func NewDbftConsensus(Blockchain *blockchain.BlockChain) *DbftConsensus {
	return &DbftConsensus{
		Blockchain:   Blockchain,
		BlockManager: blockchain.NewBlockManager(),
		VoteResults:  blockchain.NewVoteResults(),
		Locker:       sync.RWMutex{},
	}
}

func (dbft DbftConsensus) GetRound() *types.Round {
	return dbft.Round
}

// 校验从其他委托人节点过来的区块数据
func (dbft DbftConsensus) BlockFromPeer(ctxlog *ctxlog.ContextLog, block blockchain.Block, header blockchain.Header) {
	dbft.Locker.Lock()
	defer dbft.Locker.Unlock()

	dbft.BlockManager.Insert(header)

	status := dbft.BlockManager.GetBlockStatus(header.CaculateHash())
	if status == blockchain.BLOCK_SAVED ||
		(status > blockchain.BLOCK_ERROR_START && status < blockchain.BLOCK_ERROR_END) ||
		status == blockchain.BLOCK_VOTED {
		ctxlog.Log("status", status)
		//如果区块已经写入链中 or 是一个有问题的区块 or 已经投票成功 直接返回
		return
	}

	if status == blockchain.BLOCK_VALID {
		ctxlog.Log("SendVote", true)
		dbft.SendVote(header)
		dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_VOTED)
		return
	}

	// 判断此区块是否是一个interval之前打包的，如果是则放弃vote
	// unit： ms    单位：ms
	blockLatencyTime := int(time.Now().UnixNano()/1e6 - header.Timestamp) // 从节点打包到当前节点的延迟，单位ms
	blockInterval := int(blockchain.BackboneBlockInterval / 1e6)          // 当前链的打包间隔，单位nanoSecond,计算为ms
	if blockLatencyTime > blockInterval {
		ctxlog.Log("More than an interval", true)
		dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_ERROR_BROADCAST_TIME)
		return
	}

	if !dbft.ValidatePackRight(header.Timestamp, dbft.Blockchain.LastBlock().Timestamp, dbft.Round, block.Miner) {
		ctxlog.Log("Invalid node", true)
		return
	}

	transactions := block.GetTransactions()
	receipts := block.GetTxReceipts()
	// 对区块进行validate和recover，如果区块数据没问题，则发送投票给其他节点
	if dbft.Blockchain.LastBlock().ValidateBlockStat(header, transactions, receipts) {
		ctxlog.Log("SendVote", true)
		dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_VOTED)
		dbft.SendVote(header)
	} else {
		ctxlog.Log("error body", true)
		dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_ERROR_BODY)
	}
}

func (dbft DbftConsensus) getUserEvent(peer types.Peer, eventId string) (userevent.IUserEvent, error) {
	data, err := peer.GetDBValue(eventId)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	eventType, exist := m["EventType"]
	if exist {
		t, ok := eventType.(string)
		if !ok {
			return nil, errors.New("Invalid event type")
		}
		switch t {
		case userevent.TYPE_USEREVENT_PUBLIC_TOKEN:
			var tokenIssue userevent.TokenIssue
			err = json.Unmarshal(data, &tokenIssue)
			if err != nil {
				return nil, err
			}
			if !userevent.Validate(&tokenIssue) {
				return nil, errors.New("Invalid signature")
			}
			if !strings.EqualFold(eventId, tokenIssue.EventId()) {
				return nil, errors.New("Invalid hash")
			}
			return &tokenIssue, nil
		case userevent.TYPE_USEREVENT_TRANSACTION:
			var tx userevent.Transaction
			err = json.Unmarshal(data, &tx)
			if err != nil {
				return nil, err
			}
			if !userevent.Validate(&tx) {
				return nil, errors.New("Invalid signature")
			}
			return &tx, nil
		default:
			return nil, errors.New("Invalid event type")
		}
	} else {
		return nil, errors.New("Invalid event type")
	}
}

//func (dbft DbftConsensus) getBlockEvents(block *blockchain.Header) (list []userevent.IUserEvent, err error) {
//	if !dbft.syncBlockBody(block) {
//		return nil, errors.New("synchronized body error")
//	}
//	eventIds := []string{}
//	//eventIds := block.BlockBody.Events
//	if len(eventIds) > 0 {
//		for _, eventId := range eventIds {
//			eventGetter := pool.NewEventGetter(eventId)
//			dbft.Blockchain.Pool.EventGetter <- eventGetter
//			event := <-eventGetter.Chan
//			if event == nil {
//				event, err = dbft.getUserEvent(block.GetRound().Peers[block.GetRound().CurrentIndex], eventId)
//				if err != nil {
//					log.Crit("Invalid EventId %s, error: %s", eventId, err.Error())
//					return nil, err
//				}
//				dbft.Blockchain.Pool.EventPutter <- event
//			}
//			list = append(list, event)
//		}
//	}
//	return list, nil
//}

//func (dbft DbftConsensus) syncBlockBody(block *blockchain.Header) bool {
//	// 从打包节点获取body
//	body, err := block.GetRound().Peers[block.GetRound().CurrentIndex].GetDBValue(hex.EncodeToString(block.Body))
//	if err != nil {
//		return false
//	}
//	block.BlockBody, err = blockchain.FromBytes2BLockBody(body)
//	if err != nil {
//		return false
//	}
//	return true
//}

// 校验从其他委托人节点来的区块成功之后发送投票
func (dbft DbftConsensus) SendVote(header blockchain.Header) {
	// 同一个节点在一个出块interval内对一个高度只会投票一次，所以先校验是否进行过投票
	//log.Info("Validating send vote interval.")
	// 获取上次投票时间 lastVoteTime < 0 表明当前区块没有投票过
	lastVoteTime := dbft.BlockManager.GetVoteTime(header.Height)
	if lastVoteTime > 0 {
		// 距离投票的毫秒数
		intervalInFact := int(time.Now().UnixNano()/1e6 - lastVoteTime)
		// 规则指定的毫秒数
		intervalInRule := int(blockchain.BackboneBlockInterval / 1e6)

		// 说明在一个intervalInRule内进行过投票
		if intervalInFact < intervalInRule {
			log.Info("This height has voted in paste interval, return.")
			return
		}
	}

	// 记录此次投票的时间
	dbft.BlockManager.SetVoteTime(header.Height, time.Now().UnixNano()/1e6)

	// 生成vote对象
	vote := &blockchain.BlockVote{
		BlockchainId: dbft.Blockchain.ChainId,
		BlockHash:    header.CaculateHash(),
		BlockHeight:  header.Height,
		VoteResult:   true,
		Peer:         conf.EKTConfig.Node,
	}

	// 签名
	err := vote.Sign(conf.EKTConfig.GetPrivateKey())
	if err != nil {
		log.Crit("Sign vote failed, recorded. %v", err)
		return
	}

	// 向其他节点发送签名后的vote信息
	log.Info("Signed this vote, sending vote result to other peers.")
	for _, peer := range dbft.Round.Peers {
		url := fmt.Sprintf(`http://%s:%d/vote/api/vote`, peer.Address, peer.Port)
		go util.HttpPost(url, vote.Bytes())
	}
	log.Info("Send vote to other peer succeed.")
}

// for循环+recover保证DBFT线程的安全性
func (dbft *DbftConsensus) StableRun() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.PrintStack("dbft.StableRun")
				}
			}()
			dbft.Run()
		}()
	}
}

// 委托人节点检测其他节点未按时出块的情况下， 当前节点进行打包的逻辑
func (dbft DbftConsensus) delegateRun() {
	log.Info("DBFT started.")

	//要求有半数以上节点存活才可以进行打包区块
	moreThanHalf := false
	for !moreThanHalf {
		if AliveDelegatePeerCount(param.MainChainDelegateNode, false) <= len(param.MainChainDelegateNode)/2 {
			log.Info("Alive node is less than half, waiting for other delegate node restart.")
			time.Sleep(3 * time.Second)
		} else {
			moreThanHalf = true
		}
	}

	// 每1/4个interval检测一次是否有漏块，如果发生漏块且当前节点可以出块，则进入打包流程
	interval := blockchain.BackboneBlockInterval / 4
	for {
		// 判断是否是当前节点打包区块

		if dbft.IsMyTurn() {
			dbft.Pack(ctxlog.NewContextLog("Pack signal from delegate thread."))
		}

		time.Sleep(interval)
	}
}

func (dbft DbftConsensus) ValidatePackRight(packTime, lastBlockTime int64, lastRound *types.Round, node types.Peer) bool {
	if lastRound == nil {
		return param.MainChainDelegateNode[0].Equal(node)
	} else {
		round := lastRound.Clone()
		intervalInFact, interval := int(packTime-lastBlockTime), int(blockchain.BackboneBlockInterval/1e6)

		// n表示距离上次打包的间隔
		n := int(intervalInFact) / int(interval)
		remainder := int(intervalInFact) % int(interval)
		if n == 0 {
			if remainder < int(interval)*3/2 {
				return round.Peers[(round.CurrentIndex+1)%round.Len()].Equal(node)
			} else {
				return false
			}
		}
		n++
		return round.Peers[(round.CurrentIndex+n)%round.Len()].Equal(node)
	}
}

// 判断peer在指定时间是否有打包区块的权力
func (dbft DbftConsensus) PeerTurn(packTime, lastBlockTime int64, peer types.Peer) bool {
	log.Info("Validating peer has the right to pack block.")

	// 如果当前高度为0，则需要第一个节点进行打包
	if dbft.Blockchain.GetLastHeight() == 0 {
		if dbft.GetRound().Peers[0].Equal(peer) {
			return true
		} else {
			return false
		}
	}

	intervalInFact, interval := int(packTime-lastBlockTime), int(blockchain.BackboneBlockInterval/1e6)

	// n表示距离上次打包的间隔
	n := int(intervalInFact) / int(interval)
	remainder := int(intervalInFact) % int(interval)

	// 打包的下半个周期不再开始打包
	if remainder > int(interval)/2 {
		return false
	}

	// 如果距离上次打包在一个interval之内，返回false，通过vote触发进入打包阶段
	if n == 0 {
		return false
	}

	// 超过n个interval则需要第n+1个节点进行打包
	n++

	// 判断peer是否拥有打包权限
	_round := dbft.GetRound().Clone()
	return _round.Peers[(_round.CurrentIndex+n)%_round.Len()].Equal(peer)
}

// 用于委托人线程判断当前节点是否有打包权限
func (dbft DbftConsensus) IsMyTurn() bool {
	now := time.Now().UnixNano() / 1e6
	lastPackTime := dbft.Blockchain.LastBlock().Timestamp
	result := dbft.PeerTurn(now, lastPackTime, conf.EKTConfig.Node)

	return result
}

func (dbft *DbftConsensus) Run() {

	log.Info("Synchronizing blockchain ...")
	interval, failCount := 50*time.Millisecond, 0
	// 同步区块
	for height := dbft.Blockchain.GetLastHeight() + 1; ; {
		log.Info("Synchronizing block at height %d.", height)
		if dbft.SyncHeight(height) {
			log.Info("Synchronizing block at height %d successed. \n", height)
			height++
			// 同步成功之后，failCount变成0
			failCount = 0
		} else {
			failCount++
			// 如果区块同步失败，会重试三次，三次之后判断当前节点是否是委托人节点，选择不同的同步策略
			if failCount >= 3 {
				log.Info("Fail count more than 3 times.")
				// 如果当前节点是委托人节点，则不再根据区块高度同步区块，而是通过投票结果来同步区块
				for _, peer := range param.MainChainDelegateNode {
					if peer.Equal(conf.EKTConfig.Node) {
						log.Info("This peer is Delegate node, start delegate thread.")
						// 开启Delegate线程并让此线程sleep
						dbft.startDelegateThread()
						<-make(chan bool)
					}
				}
				log.Info("Synchronize interval change to blockchain interval")
				interval = blockchain.BackboneBlockInterval
			}
		}
		time.Sleep(interval)
	}
}

// 开启delegate线程
func (dbft *DbftConsensus) startDelegateThread() {
	// 稳定启动dbft.delegateRun()
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.PrintStack("dbft.startDelegateThread.delegateRun")
					}
				}()
				dbft.delegateRun()
			}()
		}

	}()

	// 稳定启动dbft.delegateSync()
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.PrintStack("dbft.startDelegateThread.delegateSync")
					}
				}()
				dbft.delegateSync()
			}()
		}

	}()
}

// delegateSync同步主要是监控在一定interval如果height没有被委托人间投票改变，则通过height进行同步
func (dbft *DbftConsensus) delegateSync() {
	lastHeight := dbft.Blockchain.GetLastHeight()
	for {
		height := dbft.Blockchain.GetLastHeight()
		if height == lastHeight {
			log.Debug("Height has not change for an interval, synchronizing block.")
			func() {
				defer func() {
					if r := recover(); r != nil {
						var buf [4096]byte
						runtime.Stack(buf[:], false)
						log.Crit("Panic occured at delegate sync thread, %s", string(buf[:]))
					}
				}()
				if dbft.SyncHeight(lastHeight + 1) {
					log.Debug("Synchronized block at lastHeight %d.", lastHeight+1)
					lastHeight = dbft.Blockchain.GetLastHeight()
				} else {
					log.Debug("Synchronize block at lastHeight %d failed.", lastHeight+1)
					time.Sleep(blockchain.BackboneBlockInterval)
				}
			}()
		}

		lastHeight = dbft.Blockchain.GetLastHeight()
	}
}

// 共识向blockchain发送signal进行下一个区块的打包
func (dbft DbftConsensus) Pack(ctxlog *ctxlog.ContextLog) {
	// 对下一个区块进行打包
	lastBlock := dbft.Blockchain.LastBlock()
	dbft.Locker.Lock()
	if dbft.BlockManager.CheckHeightInterval(lastBlock.Height+1, int64(blockchain.BackboneBlockInterval)) {
		dbft.BlockManager.SetBlockStatusByHeight(lastBlock.Height+1, time.Now().UnixNano())
		dbft.Locker.Unlock()
	} else {
		dbft.Locker.Unlock()
		return
	}
	block := dbft.Blockchain.PackSignal(ctxlog, lastBlock.Height+1)

	// 如果block不为空，说明打包成功，签名后转发给其他节点
	if block != nil {
		// 计算hash
		block.CaculateHash()

		log.Debug("Packed a block at height %d, block info: %s .\n", dbft.Blockchain.GetLastHeight()+1, string(block.Bytes()))

		// 增加打包信息
		dbft.BlockManager.Insert(*block)
		dbft.BlockManager.SetBlockStatus(block.CaculateHash(), blockchain.BLOCK_VALID)
		dbft.BlockManager.SetBlockStatusByHeight(block.Height, block.Timestamp)

		// 签名
		if err := block.Sign(conf.EKTConfig.PrivateKey); err != nil {
			log.Crit("Sign block failed. %v", err)
		} else {
			// 广播
			dbft.broadcastBlock(block)
			ctxlog.Log("block", block)
		}
	}
}

// 广播区块
func (dbft DbftConsensus) broadcastBlock(block *blockchain.Header) {
	log.Info("Broadcasting block to the other peers.")
	data := block.Bytes()
	for _, peer := range dbft.GetRound().Peers {
		url := fmt.Sprintf(`http://%s:%d/block/api/newBlock`, peer.Address, peer.Port)
		go util.HttpPost(url, data)
	}
}

// 从db中recover数据
func (dbft DbftConsensus) RecoverFromDB() {
	header := encapdb.GetLastHeader(dbft.Blockchain.ChainId)
	// 如果是第一次打开
	if header == nil {
		// 将创世块写入数据库
		accounts := conf.EKTConfig.GenesisBlockAccounts
		block := blockchain.CreateGenesisBlock(accounts)
		header = block.Header
		encapdb.SetLastHeader(dbft.Blockchain.ChainId, *header)
		encapdb.SetHeaderByHeight(dbft.Blockchain.ChainId, block.Header.Height, *header)
	}
	dbft.Blockchain.SetLastBlock(*header)
	log.Info("Recovered from local database.")
}

// 获取存活的委托人节点数量
func AliveDelegatePeerCount(peers types.Peers, print bool) int {
	count := 0
	for _, peer := range peers {
		if peer.IsAlive() {
			if print {
				log.Info("Peer %s is alive, address: %s \n", peer.Account, peer.Address)
			}
			count++
		}
	}
	return count
}

// 根据height同步区块
func (dbft DbftConsensus) SyncHeight(height int64) bool {
	log.Info("Synchronizing block at height %d \n", height)
	if dbft.Blockchain.GetLastHeight() >= height {
		return true
	}
	peers := param.MainChainDelegateNode
	for _, peer := range peers {
		// 为超级节点进行判断，如果该节点是自己则跳过, 减小同步耗时
		if peer.Equal(conf.EKTConfig.Node) {
			continue
		}
		block, err := getBlockHeader(peer, height)
		if err != nil {
			log.Info("Get block header by height failed, block=%v, err=%v. \n", block, err)
			continue
		}
		if block.Height != height {
			log.Crit("Invalid height from %s, want %d, get %s \n", peer.Address, height, string(block.Bytes()))
			continue
		}
		votes, err := getVotes(peer, hex.EncodeToString(block.CaculateHash()))
		if err != nil {
			log.Info("Error peer has no votes. %v", err)
			continue
		}
		if votes.Validate() {
			//events, err := dbft.getBlockEvents(block)
			//if err != nil {
			//	continue
			//}
			//if len(events) == 0 {
			//	log.Info("It's a blank block. Don't need to validate next block.")
			//	if dbft.RecieveVoteResultWhenBlank(votes, *block) {
			//		return true
			//	} else {
			//		continue
			//	}
			//}
			//if dbft.Blockchain.LastBlock().ValidateNextBlock(*block, events) {
			//	dbft.SaveHeader(block, votes)
			//}
		}
	}
	return false
}

// 从其他委托人节点发过来的区块的投票进行记录
func (dbft DbftConsensus) VoteFromPeer(vote blockchain.BlockVote) {
	dbft.VoteResults.Insert(vote)

	if dbft.VoteResults.Number(vote.BlockHash) > len(dbft.GetRound().Peers)/2 {
		log.Info("Vote number more than half node, sending vote result to other nodes.")
		votes := dbft.VoteResults.GetVoteResults(hex.EncodeToString(vote.BlockHash))
		for _, peer := range dbft.GetRound().Peers {
			url := fmt.Sprintf(`http://%s:%d/vote/api/voteResult`, peer.Address, peer.Port)
			go util.HttpPost(url, votes.Bytes())
		}
	}
}

// 收到从其他节点发送过来的voteResult，校验之后可以写入到区块链中
func (dbft DbftConsensus) RecieveVoteResult(votes blockchain.Votes) bool {
	if !dbft.ValidateVotes(votes) {
		log.Info("Votes validate failed. %v", votes)
		return false
	}

	status := dbft.BlockManager.GetBlockStatus(votes[0].BlockHash)

	// 已经写入到链中
	if status == blockchain.BLOCK_SAVED {
		return true
	}

	// 未同步区块body
	if status > blockchain.BLOCK_ERROR_START && status < blockchain.BLOCK_ERROR_END {
		// 未同步区块体通过sync同步区块
		log.Crit("Invalid block and votes, block.hash = %s", hex.EncodeToString(votes[0].BlockHash))
	}

	// 区块已经校验但未写入链中
	if status == blockchain.BLOCK_VALID || status == blockchain.BLOCK_VOTED {
		dbft.SaveVotes(votes)
		header, exist := dbft.BlockManager.GetBlock(votes[0].BlockHash)
		if !exist {
			return false
		}
		encapdb.SetHeaderByHeight(dbft.Blockchain.ChainId, header.Height, header)
		dbft.Blockchain.NotifyPool(header)
		return true
	}

	return false
}

func (dbft DbftConsensus) SaveBlock(block *blockchain.Block, votes blockchain.Votes) {
	encapdb.SetVoteResults(dbft.Blockchain.ChainId, hex.EncodeToString(block.Hash), votes)
	encapdb.SetBlockByHeight(dbft.Blockchain.ChainId, block.Header.Height, *block)
	encapdb.SetHeaderByHeight(dbft.Blockchain.ChainId, block.Header.Height, *block.Header)
	dbft.Blockchain.SetLastBlock(*block.Header)
}

// 校验voteResults
func (dbft DbftConsensus) ValidateVotes(votes blockchain.Votes) bool {
	if !votes.Validate() {
		return false
	}

	if votes.Len() <= dbft.GetRound().Len()/2 {
		return false
	}
	return true
}

// 保存voteResults，用于同步区块时的校验
func (dbft DbftConsensus) SaveVotes(votes blockchain.Votes) {
	dbKey := []byte(fmt.Sprintf("block_votes:%s", hex.EncodeToString(votes[0].BlockHash)))
	db.GetDBInst().Set(dbKey, votes.Bytes())
}

// 根据区块hash获取votes
func (dbft DbftConsensus) GetVotes(blockHash string) blockchain.Votes {
	dbKey := []byte(fmt.Sprintf("block_votes:%s", blockHash))
	data, err := db.GetDBInst().Get(dbKey)
	if err != nil {
		return nil
	}
	var votes blockchain.Votes
	err = json.Unmarshal(data, &votes)
	if err != nil {
		return nil
	}
	return votes
}

//获取当前的peers
func (dbft DbftConsensus) GetCurrentDelegatePeers() types.Peers {
	return param.MainChainDelegateNode
}

// 根据height获取blockHeader
func getBlockHeader(peer types.Peer, height int64) (*blockchain.Header, error) {
	url := fmt.Sprintf(`http://%s:%d/block/api/blockByHeight?height=%d`, peer.Address, peer.Port, height)
	body, err := util.HttpGet(url)
	if err != nil {
		return nil, err
	}
	var resp x_resp.XRespBody
	err = json.Unmarshal(body, &resp)
	data, err := json.Marshal(resp.Result)
	if err == nil {
		var block blockchain.Header
		err = json.Unmarshal(data, &block)
		return &block, err
	}
	return nil, err
}

// 根据hash向委托人节点获取votes
func getVotes(peer types.Peer, blockHash string) (blockchain.Votes, error) {
	url := fmt.Sprintf(`http://%s:%d/vote/api/getVotes?hash=%s`, peer.Address, peer.Port, blockHash)
	body, err := util.HttpGet(url)
	if err != nil {
		return nil, err
	}
	var resp x_resp.XRespBody
	err = json.Unmarshal(body, &resp)
	if err == nil && resp.Status == 0 {
		var votes blockchain.Votes
		data, err := json.Marshal(resp.Result)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &votes)
		return votes, err
	}
	return nil, err
}
