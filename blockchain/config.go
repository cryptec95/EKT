package blockchain

import (
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/p2p"
)

type ChainConfig struct {
	DbPath          string
	BootNodes       p2p.Peers
	GenesisAccounts []types.Account
}
