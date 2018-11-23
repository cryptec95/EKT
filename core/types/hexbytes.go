package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

type HexBytes []byte

func (hexBytes *HexBytes) UnmarshalJSON(data []byte) error {
	data = bytes.Trim(data, `"`)
	bytes, err := hex.DecodeString(string(data))
	*hexBytes = bytes
	return err
}

func (hexBytes HexBytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, hex.EncodeToString(hexBytes))), nil
}
