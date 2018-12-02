package schema

import "fmt"

func GetHeaderByHeightKey(chainId, height int64) []byte {
	return []byte(fmt.Sprint(`GetHeaderByHeight: _%d_%d`, chainId, height))
}

func LastHeaderKey(chainId int64) []byte {
	return []byte(fmt.Sprintf("CurrentHeaderKey_%d", chainId))
}

func GetBlockByHeightKey(chainId, height int64) []byte {
	return []byte(fmt.Sprint(`GetBlockByHeight: _%d_%d`, chainId, height))
}
