package conf

import (
	"encoding/json"
	"io/ioutil"

	"github.com/EducationEKT/EKT/core/types"
)

type EKTConf struct {
	Version              string          `json:"version"`
	DBPath               string          `json:"dbPath"`
	LogPath              string          `json:"logPath"`
	Debug                bool            `json:"debug"`
	Node                 types.Peer      `json:"node"`
	BlockchainManagePwd  string          `json:"blockchainManagePwd"`
	GenesisBlockAccounts []types.Account `json:"genesisBlock"`
	PrivateKey           types.HexBytes  `json:"privateKey"`
	Env                  string          `json:"env"`
}

var EKTConfig *EKTConf

func InitConfig(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &EKTConfig)
	return err
}

func (conf EKTConf) GetPrivateKey() []byte {
	return conf.PrivateKey
}
