package encapdb

import (
	"encoding/json"

	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/db"
	"github.com/EducationEKT/EKT/log"
	"github.com/EducationEKT/EKT/schema"
)

func GetVoteResults(chainId int64, hash string) blockchain.Votes {
	key := schema.GetVoteResultsKey(chainId, hash)
	data, err := db.GetDBInst().Get(key)
	if err != nil {
		return nil
	}
	var votes blockchain.Votes
	log.LogErr(json.Unmarshal(data, &votes))
	return votes
}

func SetVoteResults(chainId int64, hash string, votes blockchain.Votes) {
	key := schema.GetVoteResultsKey(chainId, hash)
	data, _ := json.Marshal(votes)
	log.LogErr(db.GetDBInst().Set(key, data))
}
