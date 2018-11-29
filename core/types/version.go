package types

type Version map[string]interface{}

type IVersion interface {
	GetVersion() int64
}

func (version Version) GetVersion() int64 {
	v, exist := version["version"]
	if !exist {
		return -1
	}
	return int64(v.(float64))
}
