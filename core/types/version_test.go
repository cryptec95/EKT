package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestVersion_GetVersion(t *testing.T) {
	str := `{"height":10412,"timestamp":1541040881488,"totalFee":0,"previousHash":"82bb6dd07e800d07355208b33f20b1309041d6667e82015159c9ab4c566795c0","miner":"f6679c55bb45938dd00c2967834a79a26335066b7e816ce3ed330e8c4ceed0d1","statRoot":"07a74c3cfb94e37eb9abf996cfd2a212219a50cc6395750c2c14fb99f493bc6c","tokenRoot":"5ee68418ae374f06aa698329eb6e1c949dcb087854edf3e6f16e3854d0a9b127","txHash":"3f29168a00ae54717392b194435ddc555263c0252a6b16276345bd0d7c16328c","receiptHash":"685790ad7cbb6989c9ef98332caa1c7da8720e146382b38380098a8b48c08b9b","version":1}`
	var v Version
	err := json.Unmarshal([]byte(str), &v)
	fmt.Println(err, v.GetVersion())
}
