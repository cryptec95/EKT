package util

import (
	"fmt"
	"testing"
)

func TestStr2Int64(t *testing.T) {
	str1 := "1.001"
	fmt.Println(Str2Int64(str1, 8))

	str1 = "1.001"
	fmt.Println(Str2Int64(str1, 3))

	str1 = "12345.67890"
	fmt.Println(Str2Int64(str1, 8))

	str1 = "165885.12890100"
	fmt.Println(Str2Int64(str1, 8))
}
