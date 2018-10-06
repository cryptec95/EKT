package bancor

import (
	"encoding/json"
	"github.com/EducationEKT/EKT/core/userevent"
	"math"
)

type Bancor struct {
	CW float64

	ConnectAmount       float64
	InitConnectAmount   float64
	ConnectTokenAddress string

	TotalSmartToken   float64
	SelledSmartToken  float64
	SmartTokenAddress string
}

func NewBancor(cw int, connectAmount, smartTokenAmount, totalSmartToken float64, connectTokenAddress, smartTokenAddress string) *Bancor {
	return &Bancor{
		CW: float64(cw),

		ConnectAmount:       connectAmount,
		InitConnectAmount:   connectAmount,
		ConnectTokenAddress: connectTokenAddress,

		TotalSmartToken:   totalSmartToken,
		SelledSmartToken:  smartTokenAmount,
		SmartTokenAddress: smartTokenAddress,
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
		if b.ConnectAmount < b.InitConnectAmount {
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
		if b.TotalSmartToken < b.SelledSmartToken {
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
	amount := b.SelledSmartToken * (math.Pow(1+(ca/b.ConnectAmount), b.CW/1000000) - 1)
	b.ConnectAmount += ca
	b.SelledSmartToken += amount
	return amount
}

func (b *Bancor) Sell(amount float64) float64 {
	accuracy := int(1e5)

	total := float64(0)
	for i := 0; i < accuracy; i++ {
		amt := amount / float64(accuracy)
		total += b.sell(amt)
	}
	return total
}

func (b *Bancor) sell(amount float64) float64 {
	cnt := b.ConnectAmount * (math.Pow((b.SelledSmartToken+amount)/b.SelledSmartToken, 1000000/b.CW) - 1)
	b.SelledSmartToken -= amount
	b.ConnectAmount -= cnt
	return cnt
}
