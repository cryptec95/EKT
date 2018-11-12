package util

import "bytes"

func PendingLeft(data []byte, length int, char byte) []byte {
	if len(data) < length {
		prefix := bytes.Repeat([]byte{char}, length-len(data))
		return append(prefix, data...)
	}
	return data
}
