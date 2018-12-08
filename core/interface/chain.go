package _interface

import "github.com/EducationEKT/EKT/core/types"

type ChainReader interface {
	GetAccount(address []byte) (*types.Account, error)

	Author() []byte

	GetTimestamp() int64

	GetParent() []byte
}
