package util

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"
	"strings"
)

func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int32
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	if tmp < 0 {
		tmp = -tmp
	}
	return int(tmp)
}

func IntToBytes(n int) []byte {
	tmp := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, tmp)
	return bytesBuffer.Bytes()
}

func MoreThanHalf(n int) int {
	half := n/2 + 1
	return half
}

func Str2Int64(str string, accuracy int) int64 {
	arr := strings.Split(str, ".")
	left, err := strconv.Atoi(arr[0])
	if err != nil {
		return -1
	}
	right := 0
	if len(arr) == 2 {
		str := arr[1]
		if len(str) > accuracy {
			return -1
		}
		if len(str) < accuracy {
			str = strings.Join([]string{str, strings.Repeat("0", accuracy-len(str))}, "")
		}
		r, err := strconv.Atoi(str)
		if err != nil {
			return -1
		}
		right = r
	}
	return int64(left*int(math.Pow10(accuracy)) + right)
}
