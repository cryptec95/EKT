package schema

import "fmt"

func GetVoteResultsKey(chainId int64, hash string) []byte {
	return []byte(fmt.Sprint(`GetHeaderByHeight: _%d_%s`, chainId, hash))
}
