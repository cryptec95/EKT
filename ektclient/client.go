package ektclient

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/util"

	"github.com/EducationEKT/xserver/x_http/x_resp"
)

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
		resp := struct {
			Status int               `json:"status"`
			Msg    string            `json:"msg"`
			Header blockchain.Header `json:"result"`
		}{}
		err = json.Unmarshal(body, &resp)
		if err != nil || resp.Status != 0 {
			continue
		} else {
			return &resp.Header
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
		resp := struct {
			Status int              `json:"status"`
			Msg    string           `json:"msg"`
			Block  blockchain.Block `json:"result"`
		}{}
		err = json.Unmarshal(body, &resp)
		if err != nil || resp.Status != 0 {
			continue
		} else {
			return &resp.Block
		}
	}
	return nil
}

func (client Client) GetHeaderByHash(hash []byte) *blockchain.Header {
	for _, peer := range client.peers {
		data, err := peer.GetDBValue(hex.EncodeToString(hash))
		if err == nil && bytes.Equal(crypto.Sha3_256(data), hash) {
			return blockchain.FromBytes2Header(data)
		}
	}
	return nil
}

func (client Client) GetValueByHash(hash []byte) []byte {
	for _, peer := range client.peers {
		data, err := peer.GetDBValue(hex.EncodeToString(hash))
		if err == nil && bytes.Equal(crypto.Sha3_256(data), hash) {
			return data
		}
	}
	return nil
}

func (client Client) GetLastBlock() *blockchain.Header {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/last")
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		resp := struct {
			Status int               `json:"status"`
			Msg    string            `json:"msg"`
			Header blockchain.Header `json:"result"`
		}{}
		err = json.Unmarshal(body, &resp)
		if err != nil || resp.Status != 0 {
			continue
		} else {
			return &resp.Header
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
		var result x_resp.XRespBody
		err = json.Unmarshal(body, &result)
		if err != nil || result.Status < 0 || result.Result == nil {
			continue
		}
		data, err := json.Marshal(result.Result)
		if err != nil {
			continue
		}
		if votes := GetVotesFromResp(data); len(votes) != 0 {
			return votes
		}
	}
	return nil
}

func (client Client) BroadcastBlock(block blockchain.Block) {
	data := block.Bytes()
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/blockFromPeer")
		go util.HttpPost(url, data)
	}
}

func (client Client) SendVote(vote blockchain.PeerBlockVote) {
	data := vote.Bytes()
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/vote/api/vote")
		go util.HttpPost(url, data)
	}
}

func (client Client) SendVoteResult(votes blockchain.Votes) {
	data := votes.Bytes()
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/vote/api/voteResult")
		go util.HttpPost(url, data)
	}
}

func (client Client) GetSuggestionFee() int64 {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/transaction/api/fee")
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		var resp x_resp.XRespBody
		err = json.Unmarshal(body, &resp)
		if err != nil {
			continue
		}
		if resp.Result != nil {
			fee, ok := resp.Result.(int64)
			if ok {
				return fee
			}
		}
	}

	return 0
}

func (client Client) GetAccountNonce(address string) int64 {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/account/api/nonce?address=", address)
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		var resp x_resp.XRespBody
		err = json.Unmarshal(body, &resp)
		if err != nil {
			continue
		}
		if resp.Result != nil {
			fnonce, ok := resp.Result.(float64)
			if ok {
				str := strconv.FormatFloat(fnonce, 'f', 0, 64)
				nonce, _ := strconv.ParseInt(str, 10, 64)
				return nonce
			}
		}
	}

	return 0
}

func (client Client) SendTransaction(tx userevent.Transaction) error {
	data := tx.Bytes()

	for _, node := range client.peers {
		url := fmt.Sprintf(`http://%s:%d/transaction/api/newTransaction`, node.Address, node.Port)
		_, err := util.HttpPost(url, data)
		if err == nil {
			return nil
		}
	}

	return errors.New("send transaction failed")
}

func (client Client) GetGenesisAccounts() []types.Account {
	for _, node := range client.peers {
		url := fmt.Sprintf(`http://%s:%d/account/api/genesisAccount`, node.Address, node.Port)
		resp, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		result := struct {
			Status int             `json:"status"`
			Msg    string          `json:"msg"`
			Result []types.Account `json:"result"`
		}{}
		err = json.Unmarshal(resp, &result)
		if err != nil {
			continue
		}
		return result.Result
	}
	return nil
}

func GetVotesFromResp(body []byte) blockchain.Votes {
	var votes blockchain.Votes
	json.Unmarshal(body, &votes)
	return votes
}
