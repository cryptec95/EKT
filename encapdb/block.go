package encapdb

import (
	"encoding/json"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/encapdb/schema"
)

func GetBlockByHeight(chainId, height int64) (*blockchain.Header, error) {
	key := schema.GetBlockByHeightKey(chainId, height)
	hash, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil, err
	}
	return GetBlockByHash(hash)
}

func SetBlockByHeight(chainId, height int64, header blockchain.Header) error {
	hash := header.CaculateHash()
	db.GetDBInst().Set(hash, header.Bytes())
	key := schema.GetBlockByHeightKey(chainId, height)
	return db.GetDBInst().Set(key, hash)
}

func GetBlockByHash(hash types.HexBytes) (*blockchain.Header, error) {
	data, err := db.GetDBInst().Get(hash)
	if err != nil {
		return nil, err
	}
	var header blockchain.Header
	err = json.Unmarshal(data, &header)
	return &header, err
}

func GetLastBlock(chainId int64) (*blockchain.Header, error) {
	key := schema.LastBlockKey(chainId)
	hash, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil, err
	}
	return GetBlockByHash(hash)
}

func SetLastBlock(chainId int64, header blockchain.Header) error {
	key := schema.LastBlockKey(chainId)
	db.GetDBInst().Set(header.CaculateHash(), header.Bytes())
	return db.GetDBInst().Set(key, header.CaculateHash())
}
