package userevent

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/db"
)

const (
	FailType_SUCCESS = iota
	FailType_NO_GAS
	FailType_Invalid_NONCE
	FailType_NO_ENOUGH_AMOUNT
)

type Transactions []Transaction
type Receipts []TransactionReceipt

type Transaction struct {
	From         types.HexBytes `json:"from"`
	To           types.HexBytes `json:"to"`
	TimeStamp    int64          `json:"time"` // UnixTimeStamp
	Amount       int64          `json:"amount"`
	Fee          int64          `json:"fee"`
	Nonce        int64          `json:"nonce"`
	Data         string         `json:"data"`
	TokenAddress string         `json:"tokenAddress"`
	Sign         types.HexBytes `json:"sign"`
}

type TransactionReceipt struct {
	TxId     types.HexBytes `json:"txId"`
	Fee      int64          `json:"fee"`
	Success  bool           `json:"success"`
	FailType int            `json:"failType"`
}

type UserEventResult struct {
	EventId string `json:"txId"`
	Fee     int64  `json:"fee"`
	Success bool   `json:"success"`
	FailMsg string `json:"failMsg"`
}

func NewTransaction(from, to []byte, timestamp, amount, fee, nonce int64, data, tokenAddress string) *Transaction {
	return &Transaction{
		From:         from,
		To:           to,
		TimeStamp:    timestamp,
		Amount:       amount,
		Fee:          fee,
		Nonce:        nonce,
		Data:         data,
		TokenAddress: tokenAddress,
	}
}

func NewTransactionReceipt(tx Transaction, success bool, failType int) TransactionReceipt {
	return TransactionReceipt{
		TxId:     tx.TxId(),
		Success:  success,
		Fee:      tx.Fee,
		FailType: failType,
	}
}

func NewUserEventResult(event IUserEvent, fee int64, success bool, failMessage string) *UserEventResult {
	return &UserEventResult{
		EventId: event.EventId(),
		Fee:     fee,
		Success: success,
		FailMsg: failMessage,
	}
}

func (receipt1 TransactionReceipt) EqualsTo(receipt2 TransactionReceipt) bool {
	return receipt1.Fee == receipt2.Fee && receipt1.Success == receipt2.Success &&
		receipt1.FailType == receipt2.FailType && bytes.EqualFold(receipt1.TxId, receipt2.TxId)
}

func (tx Transaction) Type() string {
	return TYPE_USEREVENT_TRANSACTION
}

func (tx Transaction) GetNonce() int64 {
	return tx.Nonce
}

func (tx Transaction) GetSign() []byte {
	return tx.Sign
}

func (tx *Transaction) SetSign(sign []byte) {
	tx.Sign = sign
}

func (tx Transaction) Msg() []byte {
	return crypto.Sha3_256([]byte(tx.String()))
}

func (tx Transaction) GetFrom() []byte {
	return tx.From
}

func (tx Transaction) GetTo() []byte {
	return tx.To
}

func (tx Transaction) SetFrom(from []byte) {
	tx.From = from
}

func (tx Transaction) EventId() string {
	return tx.TransactionId()
}

func GetTransaction(txId []byte) *Transaction {
	txData, err := db.GetDBInst().Get(txId)
	if err != nil {
		return nil
	}
	return FromBytesToTransaction(txData)
}

func FromBytesToTransaction(data []byte) *Transaction {
	var tx Transaction
	err := json.Unmarshal(data, &tx)
	if err != nil {
		return nil
	}
	return &tx
}

func (usereventResult *UserEventResult) ToBytes() []byte {
	data, _ := json.Marshal(usereventResult)
	return data
}

func (usereventResult *UserEventResult) TxResult() (bool, string) {
	return usereventResult.Success, usereventResult.FailMsg
}

func (transactions Transactions) Len() int {
	return len(transactions)
}

func (transactions Transactions) Less(i, j int) bool {
	return strings.Compare(transactions[i].TransactionId(), transactions[j].TransactionId()) > 0
}

func (transactions Transactions) Swap(i, j int) {
	transactions[i], transactions[j] = transactions[j], transactions[i]
}

func (transactions Transactions) Bytes() []byte {
	data, _ := json.Marshal(transactions)
	return data
}

func (receipts Receipts) Bytes() []byte {
	data, _ := json.Marshal(receipts)
	return data
}

func (tx *Transaction) TransactionId() string {
	txData, _ := json.Marshal(tx)
	return hex.EncodeToString(crypto.Sha3_256(txData))
}

func (tx *Transaction) TxId() []byte {
	txData, _ := json.Marshal(tx)
	return crypto.Sha3_256(txData)
}

func (tx *Transaction) String() string {
	return fmt.Sprintf(`{"from": "%s", "to": "%s", "time": %d, "amount": %d, "fee": %d, "nonce": %d, "data": "%s", "tokenAddress": "%s"}`,
		hex.EncodeToString(tx.From), hex.EncodeToString(tx.To), tx.TimeStamp, tx.Amount, tx.Fee, tx.Nonce, tx.Data, tx.TokenAddress)
}

func (tx Transaction) Bytes() []byte {
	data, _ := json.Marshal(tx)
	return data
}
