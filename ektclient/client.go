package ektclient

import (
	"encoding/json"
	"fmt"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/util"
	"strconv"
	"xserver/x_http/x_resp"
)

type IClient interface {
	// block
	GetHeaderByHeight(height int64) *blockchain.Header
	GetBlockByHeight(height int64) *blockchain.Block
	GetLastBlock(peer types.Peer) *blockchain.Header

	// vote
	GetVotesByBlockHash(hash string) blockchain.Votes

	// delegate
	BroadcastBlock(block blockchain.Block)
	SendVote(vote blockchain.PeerBlockVote)
	SendVoteResult(votes blockchain.Votes)
}

type Client struct {
	peers []types.Peer
}

func NewClient(peers []types.Peer) IClient {
	return Client{peers: peers}
}

func (client Client) GetHeaderByHeight(height int64) *blockchain.Header {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/getHeaderByHeight?height=", strconv.Itoa(int(height)))
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		if header := blockchain.FromBytes2Header(body); header != nil {
			return header
		}
	}
	return nil
}

func (client Client) GetBlockByHeight(height int64) *blockchain.Block {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/getBlockByHeight?height=", strconv.Itoa(int(height)))
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		if block := blockchain.GetBlockFromBytes(body); block != nil {
			return block
		}
	}
	return nil
}

func (client Client) GetLastBlock(peer types.Peer) *blockchain.Header {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/last")
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		if block := blockchain.FromBytes2Header(body); block != nil {
			return block
		}
	}
	return nil
}

func (client Client) GetVotesByBlockHash(hash string) blockchain.Votes {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/vote/api/getVotes?hash=", hash)
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		if votes := GetVotesFromResp(body); len(votes) != 0 {
			return votes
		}
	}
	return nil
}

func (client Client) BroadcastBlock(block blockchain.Block) {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/blockFromPeer")
		go util.HttpPost(url, block.Bytes())
	}
}

func (client Client) SendVote(vote blockchain.PeerBlockVote) {
	for _, peer := range client.peers {
		url := fmt.Sprintf(`http://%s:%d/vote/api/vote`, peer.Address, peer.Port)
		go util.HttpPost(url, vote.Bytes())
	}
}

func (client Client) SendVoteResult(votes blockchain.Votes) {
	for _, peer := range client.peers {
		url := fmt.Sprintf(`http://%s:%d/vote/api/voteResult`, peer.Address, peer.Port)
		go util.HttpPost(url, votes.Bytes())
	}
}

func GetVotesFromResp(body []byte) blockchain.Votes {
	var resp x_resp.XRespBody
	err := json.Unmarshal(body, &resp)
	if err != nil || resp.Status != 0 {
		return nil
	}
	data, err := json.Marshal(resp.Result)
	if err == nil {
		var votes blockchain.Votes
		err = json.Unmarshal(data, &votes)
		return votes
	}
	return nil
}
