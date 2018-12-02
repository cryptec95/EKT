package schema

import "fmt"

func GetReceiptByTxHash(chainId int64, txHash string) []byte {
	return []byte(fmt.Sprint(`GetReceiptByTxHash: _%d_%s`, chainId, txHash))
}
