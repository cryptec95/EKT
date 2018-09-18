package ektclient

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/util"
	"strconv"
	"xserver/x_http/x_resp"
)

type IClient interface {
	// block
	GetBlockByHeight(height int64) *blockchain.Header
	GetLastBlock(peer types.Peer) *blockchain.Header

	// block body
	GetEventIds(hash []byte) []string
	GetEvents(eventIds []string) ([]userevent.IUserEvent, error)

	// vote
	GetVotesByBlockHash(hash string) blockchain.Votes
}

type Client struct {
	peers []types.Peer
}

func NewClient(peers []types.Peer) IClient {
	return Client{peers: peers}
}

func (client Client) GetBlockByHeight(height int64) *blockchain.Header {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/block/api/blockByHeight?height=", strconv.Itoa(int(height)))
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		if block := GetBlockFromResp(body); block != nil {
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
		if block := GetBlockFromResp(body); block != nil {
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

func (client Client) GetEventIds(hash []byte) []string {
	for _, peer := range client.peers {
		url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/db/api/getByHex?hash=", hex.EncodeToString(hash))
		body, err := util.HttpGet(url)
		if err != nil {
			continue
		}
		if !bytes.EqualFold(crypto.Sha3_256(body), hash) {
			continue
		}
		if err != nil {
			return nil
		}
	}
	return nil
}

func (client Client) GetEvents(eventIds []string) ([]userevent.IUserEvent, error) {
	events := make([]userevent.IUserEvent, 0)
	for _, eventId := range eventIds {
		event := client.GetEvent(eventId)
		if event != nil {
			events = append(events, event)
		} else {
			return nil, errors.New("transaction " + eventId + " not found")
		}
	}
	return events, nil
}

func (client Client) GetEvent(eventId string) userevent.IUserEvent {
	hash, err := hex.DecodeString(eventId)
	if err != nil {
		return nil
	}
	value, err := db.GetDBInst().Get(hash)
	if err == nil && bytes.EqualFold(crypto.Sha3_256(value), hash) {
		return userevent.FromBytesToUserEvent(value)
	} else {
		for _, peer := range client.peers {
			url := util.StringJoint("http://", peer.Address, ":", strconv.Itoa(int(peer.Port)), "/db/api/getByHex?hash=", eventId)
			body, err := util.HttpGet(url)
			if err != nil || !bytes.EqualFold(crypto.Sha3_256(body), hash) {
				continue
			} else {
				return userevent.FromBytesToUserEvent(body)
			}
		}
	}
	return nil
}

func GetBlockFromResp(body []byte) *blockchain.Header {
	var resp x_resp.XRespBody
	err := json.Unmarshal(body, &resp)
	if err != nil || resp.Status != 0 {
		return nil
	}
	data, err := json.Marshal(resp.Result)
	if err == nil {
		var block blockchain.Header
		err = json.Unmarshal(data, &block)
		return &block
	}
	return nil
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
