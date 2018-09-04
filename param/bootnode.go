package param

import (
	"github.com/EducationEKT/EKT/conf"
	"github.com/EducationEKT/EKT/p2p"
)

var mapping = make(map[string][]p2p.Peer)
var MainChainDelegateNode []p2p.Peer

func InitBootNodes() {
	mapping["mainnet"] = MainNet
	mapping["testnet"] = TestNet
	mapping["localnet"] = LocalNet
	MainChainDelegateNode = mapping[conf.EKTConfig.Env]
}
