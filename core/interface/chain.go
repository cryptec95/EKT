package _interface

import "github.com/EducationEKT/EKT/core/types"

type VMChain interface {
	ChainReader
	ChainWriter
}

type ChainReader interface {
	GetAccount(address []byte) (*types.Account, error)

	Author() []byte

	GetTimestamp() int64

	GetParent() []byte
}

type ChainWriter interface {
	ModifyContract(address, data []byte) error
}
