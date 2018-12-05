package context

import (
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/EducationEKT/EKT/crypto"
)

func TestSticker_Get(t *testing.T) {
	sticker := Sticker{
		m: &sync.Map{},
	}
	sticker.Save("hello", crypto.Sha3_256([]byte("123456")))
	value := sticker.Get("hello")
	if value == nil {
		t.FailNow()
	} else {
		bytes, ok := value.([]byte)
		if !ok {
			t.FailNow()
		} else {
			fmt.Println(hex.EncodeToString(bytes))
		}
	}
}
