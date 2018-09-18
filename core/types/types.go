package types

import (
	"encoding/json"
	"fmt"

	"bytes"
	"github.com/EducationEKT/EKT/util"
	"strings"
)

type Peer struct {
	Account        string `json:"account"`
	Address        string `json:"address"`
	Port           int32  `json:"port"`
	AddressVersion int    `json:"addressVersion"`
}

type Peers []Peer

func (peers Peers) Bytes() []byte {
	bts, _ := json.Marshal(peers)
	return bts
}

func (peer Peer) String() string {
	data, _ := json.Marshal(peer)
	return string(data)
}

func (peer Peer) IsAlive() bool {
	body, err := util.HttpGet(fmt.Sprintf(`http://%s:%d/peer/api/ping`, peer.Address, peer.Port))
	if err != nil || !bytes.Equal(body, []byte("pong")) {
		return false
	}
	return true
}

func (peer Peer) Equal(_peer Peer) bool {
	if strings.EqualFold(peer.Account, _peer.Account) && strings.EqualFold(peer.Address, _peer.Address) && peer.Port == _peer.Port && peer.AddressVersion == _peer.AddressVersion {
		return true
	}
	return false
}

func (peer Peer) GetDBValue(key string) ([]byte, error) {
	url := fmt.Sprintf(`http://%s:%d/db/api/getByHex?hash=%s`, peer.Address, peer.Port, key)
	return util.HttpGet(url)
}
