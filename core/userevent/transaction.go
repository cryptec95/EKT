package userevent

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
)

const (
	FailType_SUCCESS = iota
	FailType_OUT_OF_GAS
	FailType_CHECK_FAIL
	FailType_CONTRACT_ERROR
	FailType_INVALID_ADDRESS
	FailType_INVALID_CONTRACT_ADDRESS
	FailType_INIT_CONTRACT_ACCOUNT_FAIL
	FailType_CHECK_CONTRACT_SUBTX_ERROR
	FailType_CONTRACT_TIMEOUT
	FailType_CONTRACT_UPGRADE_REFUSED
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

	Additional string `json:"additional"`
}

type TransactionCore struct {
	From         types.HexBytes `json:"from"`
	To           types.HexBytes `json:"to"`
	Amount       int64          `json:"amount"`
	Fee          int64          `json:"fee"`
	Nonce        int64          `json:"nonce"`
	Data         string         `json:"data"`
	TokenAddress string         `json:"tokenAddress"`
}

type Transaction_V1 struct {
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

type Transaction_V2 struct {
	From         types.HexBytes `json:"from"`
	To           types.HexBytes `json:"to"`
	TimeStamp    int64          `json:"time"` // UnixTimeStamp
	Amount       int64          `json:"amount"`
	Fee          int64          `json:"fee"`
	Nonce        int64          `json:"nonce"`
	Data         string         `json:"data"`
	TokenAddress string         `json:"tokenAddress"`
	Sign         types.HexBytes `json:"sign"`

	Additional string `json:"additional"`
}

type Transaction_V3 struct {
	txData     TransactionCore `json:"txData"`
	Sign       types.HexBytes  `json:"sign"`
	Additional string          `json:"additional"`
}

type SubTransaction struct {
	Parent       types.HexBytes `json:"parent"`
	From         types.HexBytes `json:"from"`
	To           types.HexBytes `json:"to"`
	Amount       int64          `json:"amount"`
	Data         string         `json:"data"`
	TokenAddress string         `json:"tokenAddress"`
}

func NewSubTransaction(parent, from, to []byte, amount int64, data string, tokenAddress string) *SubTransaction {
	return &SubTransaction{
		Parent:       parent,
		From:         from,
		To:           to,
		Amount:       amount,
		Data:         data,
		TokenAddress: tokenAddress,
	}
}

type SubTransactions []SubTransaction

type TransactionReceipt struct {
	TxId            types.HexBytes  `json:"txId"`
	Fee             int64           `json:"fee"`
	Success         bool            `json:"success"`
	SubTransactions SubTransactions `json:"subTransactions"`
	FailType        int             `json:"failType"`
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

func ContractRefuseTx(tx Transaction) *TransactionReceipt {
	receipt := NewTransactionReceipt(tx, false, FailType_CONTRACT_ERROR)
	subTx := NewSubTransaction(tx.TxId(), tx.To, tx.From, tx.Amount, "contract refused", tx.TokenAddress)
	subTransactions := make(SubTransactions, 0)
	subTransactions = append(subTransactions, *subTx)
	receipt.SubTransactions = subTransactions
	return &receipt
}

func (receipt1 TransactionReceipt) EqualsTo(receipt2 TransactionReceipt) bool {
	return receipt1.Fee == receipt2.Fee && receipt1.Success == receipt2.Success &&
		receipt1.FailType == receipt2.FailType && bytes.EqualFold(receipt1.TxId, receipt2.TxId)
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

func (transactions Transactions) QuickInsert(transaction Transaction) Transactions {
	if len(transactions) == 0 {
		return append(transactions, transaction)
	}
	if transaction.GetNonce() < transactions[0].GetNonce() {
		list := make(Transactions, 0)
		list = append(list, transaction)
		list = append(list, transactions...)
		return list
	}
	if transaction.GetNonce() > transactions[len(transactions)-1].GetNonce() {
		return append(transactions, transaction)
	}
	for i := 0; i < len(transactions)-1; i++ {
		if transactions[i].GetNonce() < transaction.GetNonce() && transaction.GetNonce() < transactions[i+1].GetNonce() {
			list := make(Transactions, 0)
			list = append(list, transactions[:i+1]...)
			list = append(list, transaction)
			list = append(list, transactions[i+1:]...)
			return list
		}
	}
	return transactions
}

func (receipts Receipts) Bytes() []byte {
	data, _ := json.Marshal(receipts)
	return data
}

func (tx *Transaction) TransactionId() string {
	return hex.EncodeToString(tx.TxId())
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
