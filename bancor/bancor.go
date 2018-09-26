package bancor

import "math"

type Bancor struct {
	CW              float64
	ConnectAmount   float64
	TotalSmartToken float64
}

func NewBancor(cw int, connectAmount, smartTokenAmount float64) *Bancor {
	return &Bancor{
		CW:              float64(cw) / 1000000,
		ConnectAmount:   connectAmount,
		TotalSmartToken: smartTokenAmount,
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
