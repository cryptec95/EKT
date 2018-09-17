package schema

import "fmt"

func GetBlockByHeightKey(chainId, height int64) []byte {
	return []byte(fmt.Sprint(`GetBlockByHeight: _%d_%d`, chainId, height))
}

func LastBlockKey(chainId int64) []byte {
	return []byte(fmt.Sprintf("CurrentBlockKey_%d", chainId))
}
