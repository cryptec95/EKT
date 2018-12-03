package schema

import "fmt"

func GetReceiptByTxHashKey(chainId int64, txHash string) []byte {
	return []byte(fmt.Sprint(`GetReceiptByTxHashKey: _%d_%s`, chainId, txHash))
}
