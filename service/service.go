package service

type Service interface {
	Start() error
	Stop() error
}

type RPCFunc func(param interface{}) interface{}

type Func struct {
	FuncName string
	Func     RPCFunc
}

type FuncGroup struct {
	GroupName string
	Functions []Func
}
