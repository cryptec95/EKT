package context

import "sync"

type Sticker struct {
	m *sync.Map
}

func NewSticker() *Sticker {
	return &Sticker{
		m: &sync.Map{},
	}
}

func (sticker Sticker) Save(key string, value interface{}) {
	sticker.m.Store(key, value)
}

func (sticker Sticker) Get(key string) interface{} {
	value, exist := sticker.m.Load(key)
	if !exist {
		return nil
	}
	return value
}

func (sticker Sticker) GetInt64(key string) (int64, bool) {
	value, exist := sticker.m.Load(key)
	if !exist {
		return 0, false
	}
	result, ok := value.(int64)
	if !ok {
		return 0, false
	}
	return result, true
}

func (sticker Sticker) GetString(key string) (string, bool) {
	value, exist := sticker.m.Load(key)
	if !exist {
		return "", false
	}
	result, ok := value.(string)
	if !ok {
		return "", false
	}
	return result, true
}

func (sticker Sticker) GetBytes(key string) ([]byte, bool) {
	value, exist := sticker.m.Load(key)
	if !exist {
		return []byte(""), false
	}
	result, ok := value.([]byte)
	if !ok {
		return []byte(""), false
	}
	return result, true
}
