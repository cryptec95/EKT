package types

import (
	"encoding/hex"
	"github.com/EducationEKT/EKT/crypto"
	"strings"
)

type Heartbeat struct {
	Msg       HexBytes `json:"msg"`
	Node      Peer     `json:"node"`
	Signature HexBytes `json:"signature"`
}

func NewHeartbeat(node Peer) *Heartbeat {
	return &Heartbeat{
		Msg:  crypto.Sha3_256([]byte("123")),
		Node: node,
	}
}

func (beat Heartbeat) Sign(priv []byte) {
	sign, _ := crypto.Crypto(beat.Msg, priv)
	beat.Signature = sign
}

func (beat Heartbeat) Validate() bool {
	pubKey, err := crypto.RecoverPubKey(beat.Msg, beat.Signature)
	if err != nil || !strings.EqualFold(hex.EncodeToString(FromPubKeyToAddress(pubKey)), beat.Node.Account) {
		return false
	}
	return true
}
