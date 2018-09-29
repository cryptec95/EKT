package types

// Contract address contains 64 bytes
// The first 32 byte represents the founders of contract, it is create by system if the first
// The end 32 byte represents the true address of contract for a founder

type ContractAccount struct {
	Address      HexBytes         `json:"address"`
	Amount       int64            `json:"amount"`
	Gas          int64            `json:"gas"`
	CodeHash     HexBytes         `json:"codeHash"`
	ContractData []byte           `json:"data"`
	Balances     map[string]int64 `json:"balances"`
}

func NewContractAccount(address []byte, contractHash []byte) *ContractAccount {
	return &ContractAccount{
		Address:      address,
		Amount:       0,
		Gas:          0,
		Balances:     make(map[string]int64),
		CodeHash:     contractHash,
		ContractData: nil,
	}
}
