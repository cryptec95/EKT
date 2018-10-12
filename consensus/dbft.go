package consensus

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/ektclient"
	"github.com/EducationEKT/EKT/encapdb"
	"math"
	"sync"
	"time"

	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/ctxlog"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/param"
)

type DbftConsensus struct {
	Round        *types.Round
	Blockchain   *blockchain.BlockChain
	BlockManager *blockchain.BlockManager
	VoteResults  blockchain.VoteResults
	Client       ektclient.IClient
	seated       bool
	once         *sync.Once
}

func NewDbftConsensus(Blockchain *blockchain.BlockChain, client ektclient.IClient) *DbftConsensus {
	return &DbftConsensus{
		Round:        types.NewRound(param.MainChainDelegateNode, -1, 0),
		Blockchain:   Blockchain,
		BlockManager: blockchain.NewBlockManager(),
		VoteResults:  blockchain.NewVoteResults(),
		Client:       client,
		seated:       false,
		once:         &sync.Once{},
	}
}

func (dbft DbftConsensus) GetRound() types.Round {
	return dbft.Round.Clone()
}

// 校验从其他委托人节点过来的区块数据
func (dbft DbftConsensus) BlockFromPeer(clog *ctxlog.ContextLog, block *blockchain.Block) {
	dbft.BlockManager.Insert(block)

	header := block.GetHeader()

	// 判断此区块是否是一个interval之前打包的，如果是则放弃vote
	// unit： ms    单位：ms
	blockLatencyTime := int64(time.Now().UnixNano()/1e6 - header.Timestamp) // 从节点打包到当前节点的延迟，单位ms
	if blockLatencyTime > int64(blockchain.BackboneBlockInterval/1e6)*4/3 {
		clog.Log("More than an interval", true)
		return
	}

	status := dbft.BlockManager.GetBlockStatus(header.CaculateHash())
	clog.Log("status", status)
	if status == blockchain.BLOCK_SAVED ||
		(status > blockchain.BLOCK_ERROR_START && status < blockchain.BLOCK_ERROR_END) ||
		status == blockchain.BLOCK_VOTED {
		//如果区块已经写入链中 or 是一个有问题的区块 or 已经投票成功 直接返回
		return
	}

	if status == blockchain.BLOCK_VALID {
		if lastVoteTime := dbft.BlockManager.GetVoteTime(block.GetHeader().Height); lastVoteTime+int64(blockchain.BackboneBlockInterval)/1e6 > time.Now().UnixNano()/1e6 {
			clog.Log("Voted this height", true)
			return
		}
		if dbft.SendVote(*header) {
			dbft.BlockManager.SetVoteTime(block.GetHeader().Height, time.Now().UnixNano()/1e6)
			dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_VOTED)
			clog.Log("SendVote", true)
		}
		return
	}

	lastHeader := dbft.Blockchain.LastHeader()
	if !dbft.ValidatePackRight(header.Timestamp, lastHeader.Timestamp, hex.EncodeToString(lastHeader.Coinbase), block.Miner.Account) {
		clog.Log("Invalid node", true)
		dbft.BlockManager.SetBlockStatus(block.Hash, blockchain.BLOCK_ERROR_PACK_TIME)
		return
	}

	transactions := block.GetTransactions()
	receipts := block.GetTxReceipts()
	clog.Log("txs", transactions)
	clog.Log("receipts", receipts)
	// 对区块进行validate和recover，如果区块数据没问题，则发送投票给其他节点
	if dbft.Blockchain.ValidateBlock(*block) {
		if dbft.SendVote(*header) {
			dbft.BlockManager.SetVoteTime(block.GetHeader().Height, time.Now().UnixNano()/1e6)
			dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_VOTED)
			clog.Log("SendVote", true)
		}
	} else {
		clog.Log("error body", true)
		dbft.BlockManager.SetBlockStatus(header.CaculateHash(), blockchain.BLOCK_ERROR_BODY)
	}
}

// 校验从其他委托人节点来的区块成功之后发送投票
func (dbft DbftConsensus) SendVote(header blockchain.Header) bool {
	// 同一个节点在一个出块interval内对一个高度只会投票一次，所以先校验是否进行过投票
	// 获取上次投票时间 lastVoteTime < 0 表明当前区块没有投票过
	lastVoteTime := dbft.BlockManager.GetVoteTime(header.Height)
	if lastVoteTime > 0 {
		// 距离投票的毫秒数
		intervalInFact := int(time.Now().UnixNano()/1e6 - lastVoteTime)
		// 规则指定的毫秒数
		intervalInRule := int(blockchain.BackboneBlockInterval / 1e6)

		// 说明在一个intervalInRule内进行过投票
		if intervalInFact < intervalInRule {
			log.Info("This height has voted in paste interval, return. %d, %d, %d", time.Now().UnixNano()/1e6, lastVoteTime, intervalInRule)
			return false
		}
	}

	if dbft.Blockchain.GetLastHeight()+1 != header.Height {
		log.Info("This header is not right")
		return false
	}

	// 记录此次投票的时间
	dbft.BlockManager.SetVoteTime(header.Height, time.Now().UnixNano()/1e6)

	// 生成vote对象
	vote := &blockchain.PeerBlockVote{
		Vote: blockchain.BlockVoteDetail{
			BlockchainId: dbft.Blockchain.ChainId,
			BlockHash:    header.CaculateHash(),
			BlockHeight:  header.Height,
			VoteResult:   true,
		},
		Peer: conf.EKTConfig.Node,
	}

	// 签名
	err := vote.Sign(conf.EKTConfig.GetPrivateKey())
	if err != nil {
		log.Crit("Sign vote failed, recorded. %v", err)
		return false
	}

	// 向其他节点发送签名后的vote信息
	log.Info("Signed this vote, sending vote result to other peers.")
	go dbft.Client.SendVote(*vote)
	log.Info("Send vote to other peer succeed.")

	return true
}

func (dbft DbftConsensus) TryPack() bool {
	if dbft.seated {
		return false
	}

	lastHeader := dbft.Blockchain.LastHeader()

	if lastHeader.Height == 0 {
		if dbft.GetRound().Peers[0].Equal(conf.EKTConfig.Node) {
			dbft.seated = true
			go dbft.orderliness(time.Now().UnixNano() / 1e6)
			return true
		} else {
			return false
		}
	}

	//index := round.IndexOf(hex.EncodeToString(lastHeader.Coinbase))
	//myIndex := round.IndexOf(conf.EKTConfig.Node.Account)
	//distance := (myIndex + round.Len() - index) % round.Len()
	round := dbft.GetRound()
	distance := round.Distance(hex.EncodeToString(lastHeader.Coinbase), conf.EKTConfig.Node.Account)
	t := int64(distance) * int64(blockchain.BackboneBlockInterval) / 1e6

	lastTime := lastHeader.Timestamp
	roundTime := int64(time.Duration(round.Len())*blockchain.BackboneBlockInterval) / 1e6

	nextTime := lastTime + t
	for i := 0; nextTime >= time.Now().UnixNano()/1e6; i++ {
		nextTime += roundTime
	}

	dbft.seated = true
	go dbft.orderliness(nextTime)

	return true
}

func (dbft DbftConsensus) orderliness(packTime int64) {
	dbft.once.Do(func() {
		log.Debug("====!importent======%d", packTime)
		roundTime := time.Duration(dbft.GetRound().Len()) * blockchain.BackboneBlockInterval
		gap := 100 * time.Millisecond
		for {
			now := time.Now().UnixNano() / 1e6
			if now-packTime > int64(roundTime/1e6) {
				packTime += int64(roundTime / 1e6)
			}
			if int64(math.Abs(float64(now-packTime))) < int64(gap/time.Millisecond) {
				log.Debug("=======!pack signal==%d===%d===%d", now, packTime, int64(gap/time.Millisecond))
				go dbft.Pack(packTime)
				packTime += int64(roundTime / 1e6)
				time.Sleep(roundTime)
			} else {
				time.Sleep(gap)
			}
		}
	})
}

func (dbft DbftConsensus) ValidatePackRight(packTimeMs, lastBlockTimeMs int64, lastMiner, miner string) bool {
	round := dbft.GetRound()
	if lastBlockTimeMs == 0 {
		return round.Peers[0].Account == miner
	}

	distance := int64(round.Distance(lastMiner, miner))
	interval := int64(blockchain.BackboneBlockInterval / 1e6)
	roundTime := interval * int64(round.Len())

	return (packTimeMs-lastBlockTimeMs)%roundTime == distance*interval
}

func (dbft DbftConsensus) CheckPackInterval() bool {
	lastHeader := dbft.Blockchain.LastHeader()
	result := dbft.BlockManager.CheckHeightInterval(lastHeader.Height+1, int64(blockchain.BackboneBlockInterval))
	if result {
		dbft.BlockManager.SetBlockStatusByHeight(lastHeader.Height+1, time.Now().UnixNano()/1e6)
	}
	return result
}

// 进行下一个区块的打包
func (dbft DbftConsensus) Pack(packTime int64) {
	if !dbft.CheckPackInterval() {
		return
	}
	clog := ctxlog.NewContextLog("pack block")
	defer clog.Finish()

	lastHeader := dbft.Blockchain.LastHeader()
	block := blockchain.CreateBlock(lastHeader, packTime, conf.EKTConfig.Node)
	dbft.Blockchain.PackTransaction(clog, block)
	clog.Log("======b", time.Now().UnixNano()/1e6)
	clog.Log("======c", block.GetHeader().Timestamp)
	clog.Log("hash", hex.EncodeToString(block.Hash))
	clog.Log("height", block.GetHeader().Height)

	// 增加打包信息
	dbft.BlockManager.Insert(block)
	dbft.BlockManager.SetBlockStatus(block.Hash, blockchain.BLOCK_VALID)
	dbft.BlockManager.SetBlockStatusByHeight(block.GetHeader().Height, block.GetHeader().Timestamp)

	// 签名
	if err := block.Sign(conf.EKTConfig.PrivateKey); err != nil {
		log.Crit("Sign block failed. %v", err)
	} else {
		// 广播
		clog.Log("block", block)
		go dbft.Client.BroadcastBlock(*block)
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
		header = block.GetHeader()
		dbft.SaveBlock(&block, nil)
	}
	dbft.Blockchain.SetLastHeader(*header)
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
	block := dbft.Client.GetBlockByHeight(height)
	if block == nil {
		log.Info("Get block by height failed")
		return false
	}
	dbft.GetBlockHeader(block)
	if block.GetHeader() == nil || block.GetHeader().Height != height {
		log.Info("Get header by hash failed, hash = %s", hex.EncodeToString(block.Hash))
		return false
	} else {
		votes := dbft.Client.GetVotesByBlockHash(hex.EncodeToString(block.GetHeader().CaculateHash()))
		if votes == nil || !votes.Validate() {
			log.Info("Get votes by hash failed.")
			return false
		}
		if dbft.Blockchain.ValidateBlock(*block) {
			dbft.SaveBlock(block, votes)
			return true
		}
	}
	return false
}

// 从其他委托人节点发过来的区块的投票进行记录
func (dbft DbftConsensus) VoteFromPeer(vote blockchain.PeerBlockVote) {
	clog := ctxlog.NewContextLog("VoteFromPeer")
	defer clog.Finish()
	clog.Log("vote", vote)

	dbft.VoteResults.Insert(vote)
	if dbft.VoteResults.Number(vote.Vote.BlockHash) > len(dbft.GetRound().Peers)/2 {
		log.Info("Vote number more than half node, sending vote result to other nodes.")
		clog.Log("more than half", true)
		votes := dbft.VoteResults.GetVoteResults(hex.EncodeToString(vote.Vote.BlockHash))
		go dbft.Client.SendVoteResult(votes)
		clog.Log("sendResult", true)
	}
	clog.Log("finish", true)
}

// 收到从其他节点发送过来的voteResult，校验之后可以写入到区块链中
func (dbft DbftConsensus) RecieveVoteResult(votes blockchain.Votes) bool {
	clog := ctxlog.NewContextLog("Receive vote result")
	defer clog.Finish()
	if !dbft.ValidateVotes(votes) {
		clog.Log("invalid", true)
		return false
	}

	status := dbft.BlockManager.GetBlockStatus(votes[0].Vote.BlockHash)

	clog.Log("status", status)
	clog.Log("hash", hex.EncodeToString(votes[0].Vote.BlockHash))

	// 已经写入到链中
	if status == blockchain.BLOCK_SAVED {
		return true
	}

	// 未同步区块body
	if status > blockchain.BLOCK_ERROR_START && status < blockchain.BLOCK_ERROR_END {
		clog.Log("not sync", true)
		// 未同步区块体通过sync同步区块
		log.Crit("Invalid block and votes, block.hash = %s", hex.EncodeToString(votes[0].Vote.BlockHash))
	} else if status == blockchain.BLOCK_VALID || status == blockchain.BLOCK_VOTED {
		// 区块已经校验但未写入链中
		clog.Log("valid", true)
		block := dbft.BlockManager.GetBlock(votes[0].Vote.BlockHash)
		dbft.SaveBlock(block, votes)
		clog.Log("saved", true)
		go dbft.TryPack()
		return true
	}
	return false
}

func (dbft DbftConsensus) SaveBlock(block *blockchain.Block, votes blockchain.Votes) {
	header := *block.GetHeader()
	dbft.Round.UpdateIndex(block.Miner.Account)
	encapdb.SetVoteResults(dbft.Blockchain.ChainId, hex.EncodeToString(block.Hash), votes)
	encapdb.SetBlockByHeight(dbft.Blockchain.ChainId, header.Height, *block)
	encapdb.SetHeaderByHeight(dbft.Blockchain.ChainId, header.Height, header)
	encapdb.SetLastHeader(dbft.Blockchain.ChainId, header)
	dbft.Blockchain.SetLastHeader(header)
	dbft.Blockchain.NotifyPool(block.GetTransactions())
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

func (dbft DbftConsensus) GetBlockHeader(block *blockchain.Block) {
	header := block.GetHeader()
	if header == nil {
		header = dbft.Client.GetHeaderByHash(block.Hash)
		block.SetHeader(header)
	}
}
