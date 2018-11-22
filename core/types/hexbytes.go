package types

import (
	"encoding/hex"
	"errors"
	"fmt"
)

type HexBytes []byte

func (hexBytes *HexBytes) UnmarshalJSON(data []byte) error {
	if len(data) <= 2 {
		*hexBytes = make([]byte, 0)
		return errors.New("invalid HexBytes")
	}
	data = data[:len(data)-1][1:]
	bytes, err := hex.DecodeString(string(data))
	*hexBytes = bytes
	return err
}

func (hexBytes HexBytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, hex.EncodeToString(hexBytes))), nil
}
