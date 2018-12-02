package userevent

import "encoding/json"

type ReceiptDetail struct {
	Receipt     TransactionReceipt
	BlockNumber int64
	Index       int64
}

func (detail ReceiptDetail) Bytes() []byte {
	data, _ := json.Marshal(detail)
	return data
}
