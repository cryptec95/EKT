package encapdb

import (
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/encapdb/schema"
)

func GetBlockByHeight(chainId, height int64) *blockchain.Block {
	key := schema.GetBlockByHeightKey(chainId, height)
	data, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil
	}
	return blockchain.GetBlockFromBytes(data)
}

func SetBlockByHeight(chainId, height int64, block blockchain.Block) {
	key := schema.GetBlockByHeightKey(chainId, height)
	db.GetDBInst().Set(key, block.Bytes())
}

func GetHeaderByHeight(chainId, height int64) *blockchain.Header {
	key := schema.GetHeaderByHeightKey(chainId, height)
	hash, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil
	}
	return GetHeaderByHash(hash)
}

func SetHeaderByHeight(chainId, height int64, header blockchain.Header) error {
	hash := header.CalculateHash()
	db.GetDBInst().Set(hash, header.Bytes())
	key := schema.GetHeaderByHeightKey(chainId, height)
	return db.GetDBInst().Set(key, hash)
}

func GetHeaderByHash(hash types.HexBytes) *blockchain.Header {
	data, err := db.GetDBInst().Get(hash)
	if err != nil {
		return nil
	}
	return blockchain.FromBytes2Header(data)
}

func GetLastHeader(chainId int64) *blockchain.Header {
	key := schema.LastHeaderKey(chainId)
	hash, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil
	}
	return GetHeaderByHash(hash)
}

func SetLastHeader(chainId int64, header blockchain.Header) error {
	key := schema.LastHeaderKey(chainId)
	db.GetDBInst().Set(header.CalculateHash(), header.Bytes())
	return db.GetDBInst().Set(key, header.CalculateHash())
}
