package param

import (
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/core/types"
)

var mapping = make(map[string][]types.Peer)
var MainChainDelegateNode []types.Peer

func InitBootNodes() {
	mapping["mainnet"] = MainNet
	mapping["testnet"] = TestNet
	mapping["localnet"] = LocalNet
	MainChainDelegateNode = mapping[conf.EKTConfig.Env]
}
