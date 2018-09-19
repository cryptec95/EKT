package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

var VoteResultManager VoteResults

func init() {
	VoteResultManager = NewVoteResults()
}

type BlockVote struct {
	BlockchainId int64          `json:"blockchainId"`
	BlockHash    types.HexBytes `json:"blockHash"`
	BlockHeight  int64          `json:"blockHeight"`
	VoteResult   bool           `json:"voteResult"`
	Peer         types.Peer     `json:"peer"`
	Signature    types.HexBytes `json:"signature"`
}

type Votes []BlockVote

type VoteResults struct {
	broadcast   *sync.Map
	voteResults *sync.Map
}

func NewVoteResults() VoteResults {
	return VoteResults{
		broadcast:   &sync.Map{},
		voteResults: &sync.Map{},
	}
}

func (vote1 BlockVote) Equal(vote2 BlockVote) bool {
	return vote1.Peer.Equal(vote2.Peer) && vote1.BlockchainId == vote2.BlockchainId && bytes.EqualFold(vote1.BlockHash, vote2.BlockHash) &&
		vote1.BlockHeight == vote2.BlockHeight && vote1.VoteResult == vote2.VoteResult
}

func (voteResults VoteResults) GetVoteResults(hash string) Votes {
	obj, exist := voteResults.voteResults.Load(hash)
	if exist {
		return obj.(Votes)
	}
	return nil
}

func (voteResults VoteResults) SetVoteResults(hash string, votes Votes) {
	voteResults.voteResults.Store(hash, votes)
}

func (vote BlockVote) Validate() bool {
	pubKey, err := crypto.RecoverPubKey(crypto.Sha3_256(vote.Bytes()), vote.Signature)
	if err != nil {
		return false
	}
	if !strings.EqualFold(hex.EncodeToString(types.FromPubKeyToAddress(pubKey)), vote.Peer.Account) {
		return false
	}
	return true
}

func (vote *BlockVote) Sign(PrivKey []byte) error {
	signature, err := crypto.Crypto(crypto.Sha3_256(vote.Bytes()), PrivKey)
	if err != nil {
		return err
	} else {
		vote.Signature = signature
	}
	return nil
}

func (vote BlockVote) Bytes() []byte {
	data, _ := json.Marshal(vote)
	return data
}

func (voteResults VoteResults) Insert(vote BlockVote) {
	votes := voteResults.GetVoteResults(hex.EncodeToString(vote.BlockHash))
	if len(votes) > 0 {
		for _, _vote := range votes {
			if vote.Equal(_vote) {
				return
			}
		}
		votes = append(votes, vote)
	} else {
		votes = make([]BlockVote, 0)
		votes = append(votes, vote)
	}
	voteResults.SetVoteResults(hex.EncodeToString(vote.BlockHash), votes)
}

func (voteResults VoteResults) Number(blockHash []byte) int {
	votes := voteResults.GetVoteResults(hex.EncodeToString(blockHash))
	return len(votes)
}

func (voteResults VoteResults) Broadcasted(blockHash []byte) bool {
	_, exist := voteResults.broadcast.Load(hex.EncodeToString(blockHash))
	return exist
}

func (vote Votes) Len() int {
	return len(vote)
}

func (vote Votes) Swap(i, j int) {
	vote[i], vote[j] = vote[j], vote[i]
}

func (vote Votes) Less(i, j int) bool {
	return vote[i].Peer.String() < vote[j].Peer.String()
}

func (vote Votes) Bytes() []byte {
	data, _ := json.Marshal(vote)
	return data
}

func (votes Votes) Validate() bool {
	if len(votes) == 0 {
		return false
	}
	for i, vote := range votes {
		if !vote.Validate() || !vote.VoteResult {
			return false
		}
		for j, _vote := range votes {
			if i != j && vote.Equal(_vote) {
				return false
			}
		}
	}
	return true
}
