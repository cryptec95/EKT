package types

// Contract address contains 64 bytes
// The first 32 byte represents the founders of contract, it is create by system if the first
// The end 32 byte represents the true address of contract for a founder

type ContractAccount struct {
	Address      HexBytes         `json:"address"`
	CodeHash     HexBytes         `json:"codeHash"`
	Amount       int64            `json:"amount"`
	Gas          int64            `json:"gas"`
	ContractData HexBytes         `json:"data"`
	Balances     map[string]int64 `json:"balances"`
}
