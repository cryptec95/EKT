package bancor

import (
	"fmt"
	"testing"
)

func TestBancor_Buy(t *testing.T) {
	bancor1 := NewBancor(500000, 1, 1, 1e8, "", "1")
	bancor2 := NewBancor(500000, 1, 1, 1e8, "", "1")
	amount := bancor1.Buy(1000000)
	fmt.Println(amount)
	var cnt float64 = 0
	for i := 0; i < 10000; i++ {
		cnt += bancor2.Buy(100)
	}
	fmt.Println(cnt)
}

func TestBancor_Buy2(t *testing.T) {
	bancor1 := NewBancor(500000, 1, 1, 1e8, "", "1")
	amount1 := bancor1.Buy(1e5)
	fmt.Println(amount1)

	bancor2 := NewBancor(500000, 1e8, 1, 1e8, "", "1")
	amount2 := bancor2.Buy(1e5 * 1e8)
	fmt.Println(amount2)
}

func TestBancor_Sell(t *testing.T) {
	bancor := NewBancor(500000, 1, 1, 1e8, "", "1")
	amount := bancor.Buy(100000)
	fmt.Println(amount)
	fmt.Println(bancor.Sell(amount))
}

func TestBancor_Sell2(t *testing.T) {
	bancor := NewBancor(500000, 250, 1000, 1e8, "", "1")
	amount := bancor.Buy(10)
	fmt.Println(amount)
	fmt.Println(bancor.Sell(amount))
}
