package userevent

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

type TokenIssue struct {
	Token     types.Token    `json:"token"`
	From      types.HexBytes `json:"from"`
	Signature types.HexBytes `json:"signature"`
	Nonce     int64          `json:"nonce"`
	EventType string         `json:"EventType"`
}

func (event TokenIssue) GetNonce() int64 {
	return event.Nonce
}

func (event TokenIssue) Msg() []byte {
	return crypto.Sha3_256([]byte(fmt.Sprintf(`{"token": {"name": "%s", "symbol": "%s", "total": %d, "decimals": %d}, "from": "%s", "nonce": %d}`,
		event.Token.Name, event.Token.Symbol, event.Token.Total, event.Token.Decimals,
		hex.EncodeToString(event.GetFrom()), event.GetNonce())))
}

func (event TokenIssue) GetSign() []byte {
	return event.Signature
}

func (event *TokenIssue) SetSign(sign []byte) {
	event.Signature = sign
}

func (event TokenIssue) GetFrom() []byte {
	return event.From
}

func (event TokenIssue) Type() string {
	return event.EventType
}

func (event TokenIssue) EventId() string {
	data, _ := json.Marshal(event)
	return hex.EncodeToString(crypto.Sha3_256(data))
}

func FromBytes2TokenIssue(data []byte) (*TokenIssue, error) {
	var tokenIssue TokenIssue
	err := json.Unmarshal(data, &tokenIssue)
	if err != nil {
		return nil, err
	}
	tokenIssue.EventType = TYPE_USEREVENT_PUBLIC_TOKEN
	return &tokenIssue, nil
}

func (event TokenIssue) Bytes() []byte {
	data, _ := json.Marshal(event)
	return data
}
