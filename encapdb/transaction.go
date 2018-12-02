package encapdb

import (
	"encoding/json"
	"github.com/EducationEKT/EKT/core/userevent"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/schema"
)

func SaveReceiptByTxHash(chainId int64, txHash string, detail userevent.ReceiptDetail) {
	key := schema.GetReceiptByTxHashKey(chainId, txHash)
	db.GetDBInst().Set(key, detail.Bytes())
}

func GetReceiptByTxHash(chainId int64, txHash string) *userevent.ReceiptDetail {
	key := schema.GetReceiptByTxHashKey(chainId, txHash)
	data, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil
	}
	var detail userevent.ReceiptDetail
	err = json.Unmarshal(data, &detail)
	if err != nil {
		return nil
	}
	return &detail
}
