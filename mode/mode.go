package mode

import (
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/blockchain"
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/db"
)

type NodeMode interface {
	SetConfig(config conf.EKTConf)
	Start()
}

func SaveVotes(votes blockchain.Votes) {
	dbKey := []byte(fmt.Sprintf("block_votes:%s", hex.EncodeToString(votes[0].BlockHash)))
	db.GetDBInst().Set(dbKey, votes.Bytes())
}
