package bancor

import (
	"encoding/json"
	"github.com/EducationEKT/EKT/core/userevent"
	"math"
)

type Bancor struct {
	CW                  float64
	ConnectAmount       float64
	TotalSmartToken     float64
	InitConnectAmount   float64
	InitSmartToken      float64
	ConnectTokenAddress string
	SmartTokenAddress   string
}

func NewBancor(cw int, connectAmount, smartTokenAmount float64, connectTokenAddress, smartTokenAddress string) *Bancor {
	return &Bancor{
		CW:                  float64(cw) / 1000000,
		ConnectAmount:       connectAmount,
		InitConnectAmount:   connectAmount,
		TotalSmartToken:     smartTokenAmount,
		InitSmartToken:      smartTokenAmount,
		ConnectTokenAddress: connectTokenAddress,
		SmartTokenAddress:   smartTokenAddress,
	}
}

func (b *Bancor) Recover(data []byte) bool {
	var nb Bancor
	err := json.Unmarshal(data, &nb)
	if err == nil {
		*b = nb
		return true
	}
	return false
}

func (b Bancor) Data() []byte {
	data, _ := json.Marshal(b)
	return data
}

func (b *Bancor) Call(tx userevent.Transaction) (*userevent.TransactionReceipt, []byte) {
	if tx.TokenAddress == b.SmartTokenAddress {
		amount := b.Sell(float64(tx.Amount))
		if amount > b.ConnectAmount-b.InitConnectAmount {
			return userevent.ContractRefuseTx(tx), nil
		} else {
			subTx := userevent.NewSubTransaction(tx.TxId(), tx.To, tx.From, int64(amount), "From bancor contract", b.ConnectTokenAddress)
			receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
			subTransactions := make(userevent.SubTransactions, 0)
			subTransactions = append(subTransactions, *subTx)
			receipt.SubTransactions = subTransactions
			return &receipt, b.Data()
		}
	} else if tx.TokenAddress == b.ConnectTokenAddress {
		amount := b.Buy(float64(tx.Amount))
		if amount > b.TotalSmartToken-b.InitSmartToken {
			return userevent.ContractRefuseTx(tx), nil
		} else {
			subTx := userevent.NewSubTransaction(tx.TxId(), tx.To, tx.From, int64(amount), "From bancor contract", b.SmartTokenAddress)
			receipt := userevent.NewTransactionReceipt(tx, true, userevent.FailType_SUCCESS)
			subTransactions := make(userevent.SubTransactions, 0)
			subTransactions = append(subTransactions, *subTx)
			receipt.SubTransactions = subTransactions
			return &receipt, b.Data()
		}
	} else {
		return userevent.ContractRefuseTx(tx), nil
	}
}

func (b *Bancor) Buy(ca float64) float64 {
	amount := b.TotalSmartToken * (math.Pow(1+(ca/b.ConnectAmount), b.CW) - 1)
	b.ConnectAmount += ca
	b.TotalSmartToken += amount
	return amount
}

func (b *Bancor) Sell(amount float64) float64 {
	accuracy := int(1e5)

	if amount <= float64(accuracy) {
		return b.sell(amount)
	}

	total := float64(0)
	for i := 0; i < accuracy; i++ {
		amt := amount / float64(accuracy)
		total += b.sell(amt)
	}
	return total
}

func (b *Bancor) sell(amount float64) float64 {
	cnt := b.ConnectAmount * (math.Pow((b.TotalSmartToken+amount)/b.TotalSmartToken, float64(1)/b.CW) - 1)
	b.TotalSmartToken -= amount
	b.ConnectAmount -= cnt
	return cnt
}
