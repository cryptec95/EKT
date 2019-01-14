package service

const (
	PARAM_TYPE_BODY = iota
	PARAM_TYPE_INT
	PARAM_TYPE_FLOAT64
	PARAM_TYPE_STRING
)

const (
	PARAM_FROM_QUERY = iota
	PARAM_FROM_BODY
	PARAM_FROM_ALL
)

type Service interface {
	Start() error
	Stop() error
}

type RPCFunc func(params ...interface{}) interface{}

type Func struct {
	FuncName string
	Params   []Param
	Func     RPCFunc
}

type Param struct {
	Name string
	From int
	Type int
}

type FuncGroup struct {
	GroupName string
	Functions []Func
}
